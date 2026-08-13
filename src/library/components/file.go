package components

import (
	"errors"
	"fmt"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"os"
	"strings"
)

type BaseFile struct {
	baseComponents BaseComponents
}

func (c *BaseFile) SetBaseComponents(b BaseComponents) {
	c.baseComponents = b
}
func (c *BaseFile) UpdateRepo(url string, fileDir string) error {
	if err := common.ValidDownloadRef(url); err != nil {
		return err
	}
	if fileDir == "" {
		fileDir = c.baseComponents.GetDeployFromDir()
	}

	dotFile := strings.TrimRight(fileDir, "/")

	if common.SubString(url, 0, 4) != "http" {
		if url == "" {
			url = c.baseComponents.project.RepoUrl
		} else {
			url = strings.TrimRight(c.baseComponents.project.RepoUrl, "/") + "/" + strings.TrimLeft(url, "/")
		}

	}
	if _, err := os.Stat(dotFile); err != nil {
		if os.IsNotExist(err) {
			cmds := []string{}
			cmds = append(cmds, fmt.Sprintf("mkdir -p %s ", common.ShellQuote(dotFile)))
			cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(fileDir)))
			cmds = append(cmds, fmt.Sprintf("wget %s", common.ShellQuote(url)))
			cmd := strings.Join(cmds, "&&")
			_, err := c.baseComponents.runLocalCommand(cmd)
			return err
		}
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(dotFile)))
	// rm -rf * 依赖 shell 通配符展开,这里不能引用
	cmds = append(cmds, "rm -rf *")
	cmds = append(cmds, fmt.Sprintf("wget --no-check-certificate %s", common.ShellQuote(url)))
	cmd := strings.Join(cmds, "&&")
	_, err := c.baseComponents.runLocalCommand(cmd)
	return err

}

/**
 * 更新到指定commit版本
 */
func (c *BaseFile) UpdateToVersion() error {
	destination := c.baseComponents.getDeployWorkspace(c.baseComponents.task.LinkId)
	err := c.UpdateRepo(c.baseComponents.task.CommitId, destination)
	if err != nil {
		return err
	}
	if c.baseComponents.task.FileMd5 != "" {
		ss, err := c.CheckFiles(c.baseComponents.task.CommitId, destination)
		if err != nil && len(ss) == 0 {
			return errors.New("md5校验失败--" + err.Error())
		} else {
			if ss[0]["message"] != c.baseComponents.task.FileMd5 {
				return errors.New("md5校验失败--md5值不相等")
			}
		}
	}
	return nil
}

/**
 * 查询文件md5
 *
 */
func (c *BaseFile) CheckFiles(url string, fileDir string) ([]map[string]string, error) {
	if err := common.ValidDownloadRef(url); err != nil {
		return nil, err
	}
	if fileDir == "" {
		fileDir = c.baseComponents.GetDeployFromDir()
	}
	history := []map[string]string{}
	if common.SubString(url, 0, 4) != "http" {
		if url == "" {
			url = c.baseComponents.project.RepoUrl
		} else {
			url = strings.TrimRight(c.baseComponents.project.RepoUrl, "/") + "/" + strings.TrimLeft(url, "/")
		}
	}
	s := strings.Split(url, "/")
	fileName := s[len(s)-1]
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(fileDir)))
	cmds = append(cmds, fmt.Sprintf("test -f /usr/bin/md5sum && md5sum %s ", common.ShellQuote(fileName)))
	cmd := strings.Join(cmds, "&&")
	ss, err := c.baseComponents.runLocalCommand(cmd)
	if err != nil {
		return history, err
	}
	items := strings.Split(ss.Result, "\n")
	for _, item := range items {
		if strings.Index(item, " ") > -1 {
			md5 := common.SubString(item, 0, strings.Index(item, " "))
			history = append(history, map[string]string{"id": fileName, "message": md5})
		}
	}
	return history, nil
}
