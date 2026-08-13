package gopubssh

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/p2p/init_sever"
	"github.com/lnatpunblhna/gopub/src/library/p2p/server"
	"os/exec"
	"path/filepath"
	"time"
)

func CommandLocal(cmd string, to int) (ExecResult, error) {
	start := time.Now()
	timeout := time.After(time.Duration(to) * time.Second)
	execResultCh := make(chan *ExecResult, 1)
	go func() {
		execResult := LocalExec(cmd)
		execResultCh <- &execResult
	}()
	select {
	case res := <-execResultCh:
		sres := *res
		errorText := ""
		if sres.Error != nil {
			errorText += " commond  exec error.\n" + "rsult info :" + sres.Result + "\nerror info :" + sres.Error.Error()
		}
		if errorText != "" {
			return sres, errors.New(errorText)
		} else {
			return sres, nil
		}

	case <-timeout:
		// StartTime 必须给：入库时 created_at 取自它，零值会算成 -62135596800，
		// 结果就是超时这条记录在按时间过滤的页面上永远看不见
		res := ExecResult{Command: cmd, StartTime: start}
		res.fail(errors.New("cmd time out"))
		return res, errors.New("cmd time out")
	}

}
func LocalExec(cmd string) ExecResult {
	execResult := ExecResult{}
	execResult.StartTime = time.Now()
	execResult.Command = cmd
	execCommand := exec.Command("/bin/bash", "-c", cmd)
	var b bytes.Buffer
	execCommand.Stdout = &b
	var b1 bytes.Buffer
	execCommand.Stderr = &b1
	err := execCommand.Run()
	// 两路输出都保留：构建脚本把错误打到 stdout 很常见，
	// 旧实现失败时只留 stderr，排查时经常什么都看不到
	execResult.Stdout = b.String()
	execResult.Stderr = b1.String()
	execResult.EndTime = time.Now()
	if err != nil {
		execResult.Result = b1.String()
		execResult.fail(err)
		return execResult
	}
	execResult.Result = b.String()
	return execResult
}

func TransferByP2p(id string, hosts []string, user string, localFilePath string, remoteFilePath string, to int, algorithm SSHAlgorithm) ([]ExecResult, error) {
	// 用 0 长度切片起步：下面是 append 填充，若按 len(hosts) 预分配，
	// 结果里会先躺着 len(hosts) 个空壳记录，日志页面上就是一堆没有主机名的空行
	returnResult := make([]ExecResult, 0, len(hosts))
	start := time.Now()
	timeout := time.After(time.Duration(to) * time.Second)
	//创建传输任务
	s := server.CreateTask{ID: id, DispatchFiles: []string{localFilePath}, DestIPs: hosts}
	init_sever.P2pSvc.CreateTaskNoHttp(&s)
	taskInfoCh := make(chan *server.TaskInfo, 1)

	go func() {
		for {
			ss, _ := init_sever.P2pSvc.QueryTaskNoHttp(id)
			if ss.Status == server.TaskCompleted.String() {
				taskInfoCh <- ss
				break
			} else if ss.Status == server.TaskFailed.String() {
				taskInfoCh <- ss
				break
			}
			time.Sleep(100 * time.Millisecond)

		}
	}()
	select {
	case res := <-taskInfoCh:
		if res.Status == server.TaskCompleted.String() {
			e := ExecResult{}
			for ip, DispatchInfo := range res.DispatchInfos {
				e.LocalFilePath = localFilePath
				e.RemoteFilePath = remoteFilePath
				e.StartTime = DispatchInfo.StartedAt
				e.EndTime = DispatchInfo.FinishedAt
				e.Host = ip
				returnResult = append(returnResult, e)
			}
			err := TransP2pReName(id, hosts, user, localFilePath, remoteFilePath, 30, algorithm)
			return returnResult, err
		} else {
			for ip, DispatchInfo := range res.DispatchInfos {
				e := ExecResult{}
				if DispatchInfo.Status != server.TaskCompleted.String() {
					e.LocalFilePath = localFilePath
					e.RemoteFilePath = remoteFilePath
					e.StartTime = DispatchInfo.StartedAt
					e.Host = ip
					e.fail(fmt.Errorf("p2p transfer error, status=%s", DispatchInfo.Status))
				} else {
					e.LocalFilePath = localFilePath
					e.RemoteFilePath = remoteFilePath
					e.StartTime = DispatchInfo.StartedAt
					e.EndTime = DispatchInfo.FinishedAt
					e.Host = ip
				}
				returnResult = append(returnResult, e)
			}
			return returnResult, errors.New("p2p transfer error")

		}

	case <-timeout:
		// 超时的时候 DispatchInfos 拿不到，至少把每台目标机标成超时，
		// 否则这一步在日志里是完全空白的
		for _, host := range hosts {
			e := ExecResult{
				Host:           host,
				LocalFilePath:  localFilePath,
				RemoteFilePath: remoteFilePath,
				StartTime:      start,
			}
			e.fail(errors.New("p2p time out"))
			returnResult = append(returnResult, e)
		}
		return returnResult, errors.New("p2p time out")
	}
}

func TransP2pReName(id string, hosts []string, user string, localFilePath string, remoteFilePath string, to int, algorithm SSHAlgorithm) error {
	fileName := filepath.Base(localFilePath)
	filePath := init_sever.P2pSvc.Cfg.DownDir
	oldFile := filePath + fileName
	cmd := fmt.Sprintf("mv -f %s %s", oldFile, remoteFilePath)
	sshExecAgent := RemoteAgent{}
	sshExecAgent.Worker = 10
	sshExecAgent.TimeOut = 30 * time.Second
	sshExecAgent.Algorithm = algorithm
	port, _ := config.Int("SshPort")
	_, err := sshExecAgent.SshHostByKey(hosts, port, user, cmd)
	return err
}
