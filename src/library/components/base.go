package components

import (
	"fmt"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/jumpserver"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	gopubssh "github.com/lnatpunblhna/gopub/src/library/ssh"
	"github.com/lnatpunblhna/gopub/src/models"
	"regexp"
	"strings"
	"time"
)

const SSHTIMEOUT = 3600
const SSHWorker = 10

const SSHREMOTETIMEOUT = 600

type BaseComponents struct {
	name    string
	project *models.Project
	task    *models.Task

	// 记录归属：scope 区分上线 / 检测 / 刷新等，operatorId 用于把
	// 没有上线单的操作按人隔离，recordOff 则完全关闭写库（只读查询用）。
	// 详见 record.go
	scope      string
	stage      uint
	attempt    int
	operatorId uint
	recordOff  bool
}

func (c *BaseComponents) SetProject(project *models.Project) {
	c.project = project

}
func (c *BaseComponents) SetTask(task *models.Task) {
	c.task = task
}

/**
* 执行本地宿主机命令
 */
func (c *BaseComponents) runLocalCommand(command string) (gopubssh.ExecResult, error) {
	id := c.SaveRecord(command)
	s, err := gopubssh.CommandLocal(command, SSHTIMEOUT)
	if err != nil {
		logger.Error("本地命令执行失败:", command, err)
	}
	c.SaveRecordRes(id, "local", []gopubssh.ExecResult{s}, err)
	return s, err

}

/**
* 执行远端目标机命令
 */
func (c *BaseComponents) runRemoteCommand(command string, hosts []string) ([]gopubssh.ExecResult, error) {
	if len(hosts) == 0 {
		hostsInfo := c.GetHosts()
		for _, info := range hostsInfo {
			hosts = append(hosts, info.AllHost)
		}
	}
	id := c.SaveRecord(command)
	sshExecAgent := gopubssh.RemoteAgent{}
	sshExecAgent.Worker = SSHWorker
	sshExecAgent.TimeOut = time.Duration(SSHREMOTETIMEOUT) * time.Second
	sshExecAgent.Algorithm = gopubssh.SSHAlgorithm(c.project.SshAlgorithm)
	port, _ := config.Int("SshPort")
	s, err := sshExecAgent.SshHostByKey(hosts, port, c.project.ReleaseUser, command)
	if err != nil {
		logger.Error("远端命令执行失败:", hosts, err)
	}
	c.SaveRecordRes(id, "remote", s, err)
	return s, err

}

/**
* 执行远端传输文件
 */
func (c *BaseComponents) copyFilesBySftp(src string, dest string, hosts []string) ([]gopubssh.ExecResult, error) {
	if len(hosts) == 0 {
		hostsInfo := c.GetHosts()
		for _, info := range hostsInfo {
			hosts = append(hosts, info.AllHost)
		}
	}
	id := c.SaveRecord("Transfer " + src + " -> " + dest)
	sshExecAgent := gopubssh.RemoteAgent{}
	sshExecAgent.Worker = SSHWorker
	sshExecAgent.TimeOut = time.Duration(SSHREMOTETIMEOUT) * time.Second
	sshExecAgent.Algorithm = gopubssh.SSHAlgorithm(c.project.SshAlgorithm)
	port, _ := config.Int("SshPort")
	s, err := sshExecAgent.SftpHostByKey(hosts, port, c.project.ReleaseUser, src, dest)
	if err != nil {
		logger.Error("文件传输失败:", src, "->", dest, err)
	}
	c.SaveRecordRes(id, "transfer", s, err)
	return s, err

}

/**
* 执行远端传输文件 p2p方式
 */
func (c *BaseComponents) copyFilesByP2p(id string, src string, dest string, hosts []string) ([]gopubssh.ExecResult, error) {
	rid := c.SaveRecord("Transfer by p2p " + src + " -> " + dest)
	if len(hosts) == 0 {
		hostsInfo := c.GetHosts()
		for _, info := range hostsInfo {
			hosts = append(hosts, info.Ip)
		}
	}
	s, err := gopubssh.TransferByP2p(id, hosts, c.project.ReleaseUser, src, dest, SSHREMOTETIMEOUT, gopubssh.SSHAlgorithm(c.project.SshAlgorithm))
	if err != nil {
		logger.Error("p2p 传输失败:", src, "->", dest, err)
	}
	c.SaveRecordRes(rid, "transfer", s, err)
	return s, err

}

type HostInfo struct {
	Ip      string
	Group   int
	Port    int
	AllHost string
}

func (c *BaseComponents) GetHosts() []HostInfo {
	enableJumpserver, _ := config.Bool("enableJumpserver")
	if enableJumpserver == true {
		return c.GetHosts_jumpserver()
	} else {
		return c.GetHosts_database()
	}
}

func (c *BaseComponents) GetHosts_jumpserver() []HostInfo {
	hostgroupStr := c.project.HostGroup
	aGroupid := strings.Split(hostgroupStr, " ")
	res := []HostInfo{}
	port := 22
	if len(aGroupid) > 0 {
		for _, gid := range aGroupid {
			aIp2hostname, err := jumpserver.GetIpsByGroupid(string(gid))
			if err != nil {
				// 取不到就会以空主机列表继续发布，必须留痕
				logger.Error("从 jumpserver 获取节点", gid, "的资产失败:", err)
			}
			if len(aIp2hostname) > 0 {
				for ip, _ := range aIp2hostname {
					res = append(res,
						HostInfo{
							Ip:      ip,
							Port:    port,
							Group:   1,
							AllHost: ip})
				}
			}
		}
	}

	return res
}

/**
 * 获取host
 */
func (c *BaseComponents) GetHosts_database() []HostInfo {
	hostsStr := c.project.Hosts
	if c.task != nil && c.task.Hosts != "" {
		hostsStr = c.task.Hosts
	}
	//获取ip
	reg := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)\.(\d+)`)
	hosts := reg.FindAll([]byte(hostsStr), -1)
	res := []HostInfo{}
	for _, host := range hosts {
		isInList := false
		for _, r := range res {
			if r.Ip == string(host) {
				isInList = true
			}
		}
		if !isInList {
			res = append(res, HostInfo{Ip: string(host), Port: 22})
		}
	}
	//格式化端口号
	reg1 := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)\.(\d+)\:(\d+)`)
	hosts1 := reg1.FindAll([]byte(hostsStr), -1)
	for _, host := range hosts1 {
		ip := strings.Split(string(host), ":")[0]
		port := strings.Split(string(host), ":")[1]
		for i, r := range res {
			if r.Ip == ip {
				res[i].Port = common.GetInt(port)
			}
		}
	}
	//格式化端口号
	reg2 := regexp.MustCompile(`(\d+)\#(\d+)\.(\d+)\.(\d+)\.(\d+)`)
	hosts2 := reg2.FindAll([]byte(hostsStr), -1)
	for _, host := range hosts2 {
		ip := strings.Split(string(host), "#")[1]
		group := strings.Split(string(host), "#")[0]
		for i, r := range res {
			if r.Ip == ip {
				res[i].Group = common.GetInt(group)
			}
		}
	}
	for i, r := range res {
		res[i].AllHost = r.Ip
	}
	return res
}

/**
 * 获取host ip
 */
func (c *BaseComponents) GetHostIps() []string {
	hosts := []string{}
	hostsInfo := c.GetHosts()
	for _, info := range hostsInfo {
		hosts = append(hosts, info.Ip)
	}
	return hosts
}

/**
 * 获取host ip加端口
 */
func (c *BaseComponents) GetAllHost() []string {
	hosts := []string{}
	hostsInfo := c.GetHosts()
	for _, info := range hostsInfo {
		hosts = append(hosts, info.AllHost)
	}
	return hosts
}
func (c *BaseComponents) GetGroupHost() map[int]string {
	hosts := map[int]string{}
	hostsInfo := c.GetHosts()
	logger.Info(hostsInfo)
	for _, info := range hostsInfo {
		hosts[info.Group] = info.Ip + ":" + common.GetString(info.Port) + "\r\n"
	}
	return hosts
}

/**
 * 获取环境
 */
func (c *BaseComponents) getEnv() string {
	if c.project.Level == 1 {
		return "test"
	}
	if c.project.Level == 2 {
		return "simu"
	}
	if c.project.Level == 3 {
		return "prod"
	}
	return "unknow"
}

/**
 * 拼接宿主机的部署隔离工作空间
 * {deploy_from}/{env}/{project}-YYmmdd-HHiiss
 */
func (c *BaseComponents) getDeployWorkspace(version string) string {
	from := c.project.DeployFrom
	env := c.getEnv()
	project := c.GetGitProjectName(c.project.RepoUrl)
	return fmt.Sprintf("%s/%s/%s-%s", strings.TrimRight(from, "/"), strings.TrimRight(env, "/"), project, version)
}

/**
 * 获取传输宿主机tar文件路径
 *
 * {deploy_from}/{env}/{project}-YYmmdd-HHiiss.tar.gz
 */
func (c *BaseComponents) getDeployPackagePath(version string) string {
	return fmt.Sprintf("%s.tar.gz", c.getDeployWorkspace(version))
}

/**
 * 拼接宿主机的仓库目录
 * {deploy_from}/{env}/{project}
 */
func (c *BaseComponents) GetDeployFromDir() string {

	from := c.project.DeployFrom
	env := c.getEnv()
	project := c.GetGitProjectName(c.project.RepoUrl)
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(from, "/"), strings.TrimRight(env, "/"), project)
}

/**
 * 获取目标机要发布的目录
 * {webroot}
 */
func (c *BaseComponents) getTargetWorkspace() string {
	return strings.TrimRight(c.project.ReleaseTo, "/")
}

/**
 * 拼接目标机要发布的目录
 * {release_library}/{project}/{version}
 */
func (c *BaseComponents) getReleaseVersionDir(version string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.project.ReleaseLibrary, "/"), c.GetGitProjectName(c.project.RepoUrl), version)
}

/**
 * 拼接目标机要发布的打包文件路径
 * {release_library}/{project}/{version}
 */
func (c *BaseComponents) getReleaseVersionPackage(version string) string {
	return fmt.Sprintf("%s.tar.gz", c.getReleaseVersionDir(version))
}

// 根据git地址获取项目名字
func (c *BaseComponents) GetGitProjectName(gitUrl string) string {
	s := strings.Split(gitUrl, "/")
	sname := s[len(s)-1]
	snames := strings.Split(sname, `.git`)
	if snames[0] == "" {
		return "filedir"
	}
	return snames[0]
}

/**
 * 清理项目目录
 *
 */
func (c *BaseComponents) RemoveLocalProjectWorkspace() error {
	gitDir := c.GetDeployFromDir()
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("rm -rf  %s ", gitDir))
	cmd := strings.Join(cmds, "&&")
	_, err := c.runLocalCommand(cmd)
	return err
}
