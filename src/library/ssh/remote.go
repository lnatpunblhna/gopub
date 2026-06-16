package gopubssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cucued/sshexec"
	"github.com/linclin/grpool"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHAlgorithm int16

const (
	SSHAlgorithmModern SSHAlgorithm = 0
	SSHAlgorithmLegacy SSHAlgorithm = 1
)

type RemoteAgent struct {
	Worker    int
	TimeOut   time.Duration
	Algorithm SSHAlgorithm
}

func (s *RemoteAgent) SshHostByKey(hosts []string, port int, user string, cmd string) ([]sshexec.ExecResult, error) {
	return s.runHosts(hosts, func(id int, host string, auths []ssh.AuthMethod) sshexec.ExecResult {
		session := &remoteSession{
			username:  user,
			hostname:  host,
			port:      port,
			auths:     auths,
			algorithm: s.Algorithm,
		}
		return session.exec(id, cmd)
	}, cmd, "", "")
}

func (s *RemoteAgent) SftpHostByKey(hosts []string, port int, user string, localFilePath string, remoteFilePath string) ([]sshexec.ExecResult, error) {
	return s.runHosts(hosts, func(id int, host string, auths []ssh.AuthMethod) sshexec.ExecResult {
		session := &remoteSession{
			username:  user,
			hostname:  host,
			port:      port,
			auths:     auths,
			algorithm: s.Algorithm,
		}
		return session.transfer(id, localFilePath, remoteFilePath)
	}, "", localFilePath, remoteFilePath)
}

func (s *RemoteAgent) runHosts(hosts []string, run func(int, string, []ssh.AuthMethod) sshexec.ExecResult, cmd string, localFilePath string, remoteFilePath string) ([]sshexec.ExecResult, error) {
	if len(hosts) == 0 {
		log.Println("no hosts")
		return nil, errors.New("no hosts")
	}
	if s.Worker == 0 {
		s.Worker = 40
	}
	if s.TimeOut == 0 {
		s.TimeOut = 3600 * time.Second
	}
	authKeys := getAuthKeys(defaultKeyFiles())
	if len(authKeys) < 1 {
		log.Println("the user no key")
		return nil, errors.New("the user no key")
	}
	pool := grpool.NewPool(s.Worker, len(hosts), s.TimeOut)
	defer pool.Release()
	pool.WaitCount(len(hosts))
	for i := range hosts {
		count := i
		pool.JobQueue <- grpool.Job{
			Jobid: count,
			Jobfunc: func() (interface{}, error) {
				return run(count, hosts[count], authKeys), nil
			},
		}
	}

	pool.WaitAll()
	returnResult := make([]sshexec.ExecResult, len(hosts))
	errorText := ""
	for res := range pool.Jobresult {
		jobId, _ := res.Jobid.(int)
		if res.Timedout {
			returnResult[jobId].Id = jobId
			returnResult[jobId].Host = hosts[jobId]
			returnResult[jobId].Command = cmd
			returnResult[jobId].LocalFilePath = localFilePath
			returnResult[jobId].RemoteFilePath = remoteFilePath
			returnResult[jobId].Error = errors.New("ssh time out")
			errorText += "the host " + hosts[jobId] + " command exec time out."
			continue
		}
		execResult, _ := res.Result.(sshexec.ExecResult)
		returnResult[jobId] = execResult
		if execResult.Error != nil {
			errorText += "the host " + execResult.Host + " command exec error.\n" + "result info :" + execResult.Result + ".\nerror info :" + execResult.Error.Error()
		}
	}
	if errorText != "" {
		return returnResult, errors.New(errorText)
	}
	return returnResult, nil
}

type remoteSession struct {
	username  string
	hostname  string
	port      int
	auths     []ssh.AuthMethod
	algorithm SSHAlgorithm
}

func (s *remoteSession) exec(id int, command string) sshexec.ExecResult {
	result := sshexec.ExecResult{
		Id:        id,
		Host:      s.hostname,
		Command:   command,
		StartTime: time.Now(),
	}
	client, err := s.client()
	if err != nil {
		result.Error = err
		return result
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		result.Error = err
		return result
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if err := session.Run(command); err != nil {
		result.Error = err
		result.Result = stderr.String()
		return result
	}
	result.Result = stdout.String()
	result.EndTime = time.Now()
	return result
}

func (s *remoteSession) transfer(id int, localFilePath string, remoteFilePath string) sshexec.ExecResult {
	result := sshexec.ExecResult{
		Id:             id,
		Host:           s.hostname,
		LocalFilePath:  localFilePath,
		RemoteFilePath: remoteFilePath,
		StartTime:      time.Now(),
	}
	client, err := s.client()
	if err != nil {
		result.Error = err
		return result
	}
	defer client.Close()

	srcFile, err := os.Open(localFilePath)
	if err != nil {
		result.Error = err
		return result
	}
	defer srcFile.Close()

	fileInfo, err := srcFile.Stat()
	if err != nil {
		result.Error = err
		return result
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		result.Error = err
		return result
	}
	defer sftpClient.Close()

	dstFile, err := sftpClient.Create(remoteFilePath)
	if err != nil {
		result.Error = err
		return result
	}
	defer dstFile.Close()

	n, err := io.Copy(dstFile, io.LimitReader(srcFile, fileInfo.Size()))
	if err != nil {
		result.Error = err
		return result
	}
	if n != fileInfo.Size() {
		result.Error = fmt.Errorf("copy: expected %v bytes, got %d", fileInfo.Size(), n)
		return result
	}
	result.EndTime = time.Now()
	return result
}

func (s *remoteSession) client() (*ssh.Client, error) {
	return ssh.Dial("tcp", s.hostname+":"+strconv.Itoa(s.port), s.clientConfig())
}

func (s *remoteSession) clientConfig() *ssh.ClientConfig {
	algorithms := ssh.SupportedAlgorithms()
	if s.algorithm == SSHAlgorithmLegacy {
		insecure := ssh.InsecureAlgorithms()
		algorithms.KeyExchanges = appendUnique(algorithms.KeyExchanges, insecure.KeyExchanges...)
		algorithms.Ciphers = appendUnique(algorithms.Ciphers, insecure.Ciphers...)
		algorithms.MACs = appendUnique(algorithms.MACs, insecure.MACs...)
		algorithms.HostKeys = appendUnique(algorithms.HostKeys, insecure.HostKeys...)
	}
	return &ssh.ClientConfig{
		Config: ssh.Config{
			KeyExchanges: algorithms.KeyExchanges,
			Ciphers:      algorithms.Ciphers,
			MACs:         algorithms.MACs,
		},
		User:              s.username,
		Auth:              s.auths,
		HostKeyCallback:   ssh.InsecureIgnoreHostKey(),
		HostKeyAlgorithms: algorithms.HostKeys,
	}
}

func defaultKeyFiles() []string {
	home := os.Getenv("HOME")
	if home == "" {
		if userHome, err := os.UserHomeDir(); err == nil {
			home = userHome
		}
	}
	return []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_dsa"),
	}
}

func getAuthKeys(keys []string) []ssh.AuthMethod {
	methods := []ssh.AuthMethod{}
	for _, keyname := range keys {
		pkey := publicKeyFile(keyname)
		if pkey != nil {
			methods = append(methods, pkey)
		}
	}
	return methods
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := os.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func appendUnique(values []string, more ...string) []string {
	existing := make(map[string]bool, len(values)+len(more))
	for _, value := range values {
		existing[value] = true
	}
	for _, value := range more {
		if existing[value] {
			continue
		}
		values = append(values, value)
		existing[value] = true
	}
	return values
}
