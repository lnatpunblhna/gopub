package components

import (
	"fmt"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"os"
	"strings"
	"sync"
	"time"
)

// 列分支/列 tag 只读取仓库元信息,同一个项目共用一份本地仓库目录。
// 打开创建上线单页面会同时请求分支和 tag,两条 git 并发跑在同一目录会撞 index.lock,
// 所以按目录串行,并给 fetch 加一个短 TTL,避免同一次开页重复拉取远程。
var repoReadLocks sync.Map // gitDir -> *sync.Mutex
var repoFetchedAt sync.Map // gitDir -> time.Time
const repoFetchTTL = 30 * time.Second

type BaseGit struct {
	baseComponents BaseComponents
}

func repoReadLock(gitDir string) *sync.Mutex {
	mu, _ := repoReadLocks.LoadOrStore(gitDir, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

/**
 * 为只读查询(分支/tag)准备本地仓库
 *
 * 仓库不存在时才整仓 clone;已存在时只 fetch --prune,
 * 不做 checkout/reset —— 列个分支没必要动工作区。
 */
func (c *BaseGit) syncRepoForRead(gitDir string) error {
	if gitDir == "" {
		gitDir = c.baseComponents.GetDeployFromDir()
	}
	mu := repoReadLock(gitDir)
	mu.Lock()
	defer mu.Unlock()

	dotGit := strings.TrimRight(gitDir, "/") + "/.git"
	if _, err := os.Stat(dotGit); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := c.UpdateRepo("", gitDir); err != nil {
			return err
		}
		repoFetchedAt.Store(gitDir, time.Now())
		return nil
	}

	if at, ok := repoFetchedAt.Load(gitDir); ok {
		if t, ok := at.(time.Time); ok && time.Since(t) < repoFetchTTL {
			return nil
		}
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(gitDir)))
	cmds = append(cmds, "/usr/bin/env git fetch -q --all --prune --tags ")
	cmd := strings.Join(cmds, " && ")
	if _, err := c.baseComponents.runLocalCommand(cmd); err != nil {
		return err
	}
	repoFetchedAt.Store(gitDir, time.Now())
	return nil
}

func (c *BaseGit) SetBaseComponents(b BaseComponents) {
	c.baseComponents = b
}
func (c *BaseGit) UpdateRepo(branch string, gitDir string) error {
	if gitDir == "" {
		gitDir = c.baseComponents.GetDeployFromDir()
	}
	if err := common.ValidGitRef(branch); err != nil {
		return err
	}
	// 分支为空时自动探测远程默认分支(master / main 自适应),不再写死 master。
	// 这一支是有意保留的命令替换,不能引用;非空分支一律走 ShellQuote,
	// 否则 BR=<分支> 这个赋值语句本身就会被注入(校验之外的第二道防线)。
	branchExpr := common.ShellQuote(branch)
	if branch == "" {
		branchExpr = "$(git remote show origin | sed -n 's/.*HEAD branch: //p')"
	}
	dotGit := strings.TrimRight(gitDir, "/") + "/.git"
	if _, err := os.Stat(dotGit); err != nil {
		if os.IsNotExist(err) {
			cmds := []string{}
			cmds = append(cmds, fmt.Sprintf("mkdir -p %s ", common.ShellQuote(gitDir)))
			cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(gitDir)))
			cmds = append(cmds, fmt.Sprintf("/usr/bin/env git clone -q %s .", common.ShellQuote(c.baseComponents.project.RepoUrl)))
			cmds = append(cmds, fmt.Sprintf("BR=%s ", branchExpr))
			cmds = append(cmds, `/usr/bin/env git checkout -q "$BR"`)
			cmd := strings.Join(cmds, " && ")
			_, err := c.baseComponents.runLocalCommand(cmd)
			return err
		}
	}
	// 用 fetch + reset 强制对齐远程分支,避开 pull 的 merge/tracking 配置
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(gitDir)))
	cmds = append(cmds, "/usr/bin/env git fetch -q --all")
	cmds = append(cmds, fmt.Sprintf("BR=%s ", branchExpr))
	cmds = append(cmds, `/usr/bin/env git checkout -q "$BR"`)
	cmds = append(cmds, `/usr/bin/env git reset -q --hard "origin/$BR"`)
	cmd := strings.Join(cmds, " && ")
	_, err := c.baseComponents.runLocalCommand(cmd)
	return err

}

/**
 * 更新到指定commit版本
 */
func (c *BaseGit) UpdateToVersion() error {
	if err := common.ValidGitRef(c.baseComponents.task.CommitId); err != nil {
		return err
	}
	destination := c.baseComponents.getDeployWorkspace(c.baseComponents.task.LinkId)
	if err := c.UpdateRepo(c.baseComponents.task.Branch, destination); err != nil {
		return err
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	cmds = append(cmds, fmt.Sprintf("/usr/bin/env git reset -q --hard %s ", common.ShellQuote(c.baseComponents.task.CommitId)))
	cmd := strings.Join(cmds, " && ")
	_, err := c.baseComponents.runLocalCommand(cmd)
	return err
}

/**
 * 获取分支列表
 */
func (c *BaseGit) GetBranchList() ([]map[string]string, error) {
	history := []map[string]string{}
	destination := c.baseComponents.GetDeployFromDir()
	if err := c.syncRepoForRead(destination); err != nil {
		return history, err
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	// 用全名而不是 %(refname:short):short 会把 refs/remotes/origin/HEAD 缩成裸的 "origin",
	// 那样过滤不掉,列表里就会多出一个叫 origin 的假分支
	cmds = append(cmds, `/usr/bin/env git for-each-ref --format='%(refname)' refs/remotes/origin `)
	cmd := strings.Join(cmds, " && ")
	s, err := c.baseComponents.runLocalCommand(cmd)
	if err != nil {
		return history, err
	}
	remotePrefix := "refs/remotes/origin/"
	items := strings.Split(s.Result, "\n")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == remotePrefix+"HEAD" || !strings.HasPrefix(item, remotePrefix) {
			continue
		}
		item = strings.TrimPrefix(item, remotePrefix)
		history = append(history, map[string]string{"id": item, "message": item})
	}
	return history, nil
}

/**
 * 获取提交历史
 *
 */
func (c *BaseGit) GetCommitList(branch string, count int) ([]map[string]string, error) {
	if count == 0 {
		count = 20

	}
	// branch 为空时交给 UpdateRepo 探测远程默认分支,不再写死 master
	history := []map[string]string{}
	destination := c.baseComponents.GetDeployFromDir()
	if err := c.UpdateRepo(branch, destination); err != nil {
		return history, err
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	cmds = append(cmds, `/usr/bin/env git log -`+common.GetString(count)+` --pretty="%h - %an %s" `)
	cmd := strings.Join(cmds, " && ")
	s, err := c.baseComponents.runLocalCommand(cmd)
	if err != nil {
		return history, err
	}
	items := strings.Split(s.Result, "\n")
	for _, item := range items {
		if strings.Index(item, "-") > -1 {
			commitId := common.SubString(item, 0, strings.Index(item, "-")-1)
			history = append(history, map[string]string{"id": commitId, "message": item})
		}
	}
	return history, nil
}

/**
 * 获取tag记录
 *
 */
func (c *BaseGit) GetTagList(count int) ([]map[string]string, error) {
	if count == 0 {
		count = 20
	}
	history := []map[string]string{}
	destination := c.baseComponents.GetDeployFromDir()
	if err := c.syncRepoForRead(destination); err != nil {
		return history, err
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	cmds = append(cmds, `/usr/bin/env git tag -l `)
	cmd := strings.Join(cmds, " && ")
	s, err := c.baseComponents.runLocalCommand(cmd)
	if err != nil {
		return history, err
	}
	items := strings.Split(s.Result, "\n")
	for _, item := range items {
		history = append(history, map[string]string{"id": item, "message": item})
	}
	return history, nil
}

func (c *BaseGit) DiffBetweenCommits(branch string, commitIdNew string, commitIdOld string) ([]string, error) {
	if err := common.ValidGitRef(commitIdNew); err != nil {
		return nil, err
	}
	if err := common.ValidGitRef(commitIdOld); err != nil {
		return nil, err
	}
	if err := c.UpdateRepo(branch, ""); err != nil {
		return nil, err
	}
	destination := c.baseComponents.GetDeployFromDir()
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	cmds = append(cmds, `/usr/bin/env git diff --name-only  `+common.ShellQuote(commitIdNew)+` `+common.ShellQuote(commitIdOld)+` `)
	cmd := strings.Join(cmds, " && ")
	s, err := c.baseComponents.runLocalCommand(cmd)
	var files []string
	if err != nil {
		return nil, err
	} else {
		items := strings.Split(s.Result, "\n")
		for _, item := range items {
			if len(item) > 0 {
				files = append(files, item)
			}
		}
		return files, nil
	}
}

func (c *BaseGit) GetLastModifyInfo(branch string, filepath string) (map[string]string, error) {
	if err := common.ValidGitRef(branch); err != nil {
		return nil, err
	}
	// filepath 来自 git diff 的输出,仓库里完全可以存在带特殊字符的文件名,同样要引用
	destination := c.baseComponents.GetDeployFromDir()
	// branch 为空时不能引用成 '' ——那会变成一个空 pathspec,git 直接报错
	pathspec := common.ShellQuote(filepath)
	if branch != "" {
		pathspec = common.ShellQuote(branch) + " " + pathspec
	}
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	cmds = append(cmds, `/usr/bin/env git log -- `+pathspec+` | head -3 | tail -2`)
	cmd := strings.Join(cmds, " && ")
	s, err := c.baseComponents.runLocalCommand(cmd)
	if err != nil {
		return nil, err
	} else {
		lines := strings.Split(s.Result, "\n")

		name := common.SubString(lines[0], 8, 100)
		time := common.SubString(lines[1], 8, 100)

		var fileinfo map[string]string
		fileinfo = make(map[string]string)
		fileinfo["name"] = name
		fileinfo["time"] = time
		return fileinfo, nil
	}
}
