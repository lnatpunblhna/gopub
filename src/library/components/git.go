package components

import (
	"errors"
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

/**
 * 解析远程默认分支的具体名字(master / main 自适应)
 *
 * 早先这一步是内联进 && 链的 BR=$(git remote show origin | sed -n 's/.*HEAD branch: //p'),
 * 有两个坑叠在一起:
 *   - 管道的退出码取自最后一环 sed,而 sed -n 匹配不到任何行时照样返回 0,
 *     所以 git remote show origin 失败(远程不可达 / 认证失败)会被完全吞掉;
 *   - git 那行输出是可本地化的,非英文 locale 下 "HEAD branch:" 会被翻译,sed 同样匹配不到。
 * 两种情况都得到空 BR,再拼进 git checkout -q "$BR" 就是
 * fatal: empty string is not a valid pathspec(exit 128),而且完全看不出真正的原因。
 *
 * 现在优先读本地 ref:clone 时就已写好,不联网、不受 locale 影响;
 * 拿不到才回退去问一次远程,并且空结果一律当错误上报。
 */
func (c *BaseGit) resolveBranch(branch string, gitDir string) (string, error) {
	if branch != "" {
		return branch, nil
	}
	cdCmd := fmt.Sprintf("cd %s ", common.ShellQuote(gitDir))
	detected := ""
	// symbolic-ref 直接读 refs/remotes/origin/HEAD,输出形如 origin/master
	cmd := strings.Join([]string{cdCmd, `/usr/bin/env git symbolic-ref -q --short refs/remotes/origin/HEAD`}, " && ")
	if s, err := c.baseComponents.runLocalCommand(cmd); err == nil {
		detected = strings.TrimPrefix(strings.TrimSpace(s.Result), "origin/")
	}
	if detected == "" {
		// 老仓库可能没写 origin/HEAD,问一次远程兜底。
		// LC_ALL=C 固定英文输出,且这里不接管道,git 自己的失败能如实反映到退出码上。
		cmd = strings.Join([]string{cdCmd, `LC_ALL=C /usr/bin/env git remote show origin`}, " && ")
		s, err := c.baseComponents.runLocalCommand(cmd)
		if err != nil {
			return "", fmt.Errorf("探测远程默认分支失败,请检查仓库地址与访问权限,或在上线单里显式指定分支: %w", err)
		}
		detected = parseRemoteHeadBranch(s.Result)
	}
	if detected == "" {
		return "", errors.New("探测不到远程默认分支,请在上线单里显式指定分支")
	}
	// 分支名来自命令输出,仍然过一遍校验再拼进后续命令
	if err := common.ValidGitRef(detected); err != nil {
		return "", fmt.Errorf("探测到的默认分支不合法(%s): %w", detected, err)
	}
	return detected, nil
}

// parseRemoteHeadBranch 从 git remote show origin 的输出里取 HEAD branch 那一行。
// 远程没设置 HEAD 时 git 会打印 (unknown),那不是分支名,按探测失败处理。
func parseRemoteHeadBranch(out string) string {
	for _, line := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "HEAD branch:")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		if rest == "(unknown)" {
			return ""
		}
		return rest
	}
	return ""
}

func (c *BaseGit) UpdateRepo(branch string, gitDir string) error {
	if gitDir == "" {
		gitDir = c.baseComponents.GetDeployFromDir()
	}
	if err := common.ValidGitRef(branch); err != nil {
		return err
	}
	dotGit := strings.TrimRight(gitDir, "/") + "/.git"
	if _, err := os.Stat(dotGit); err != nil {
		// 只有"确实不存在"才该走 clone;其它错误(如权限)原先会穿透到下面的 fetch,
		// 报出来的是 cd 失败,掩盖真正的原因
		if !os.IsNotExist(err) {
			return err
		}
		cmds := []string{}
		cmds = append(cmds, fmt.Sprintf("mkdir -p %s ", common.ShellQuote(gitDir)))
		cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(gitDir)))
		cmds = append(cmds, fmt.Sprintf("/usr/bin/env git clone -q %s .", common.ShellQuote(c.baseComponents.project.RepoUrl)))
		// clone 完成时工作区已经在远程默认分支上,分支为空就不必再 checkout,
		// 也就不需要再去探测默认分支名;只有显式指定了分支才切一次
		if branch != "" {
			cmds = append(cmds, fmt.Sprintf("/usr/bin/env git checkout -q %s", common.ShellQuote(branch)))
		}
		cmd := strings.Join(cmds, " && ")
		_, err := c.baseComponents.runLocalCommand(cmd)
		return err
	}
	// 仓库已存在:先 fetch,再把空分支解析成具体名字,最后 checkout + reset 强制对齐远程,
	// 避开 pull 的 merge/tracking 配置
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(gitDir)))
	cmds = append(cmds, "/usr/bin/env git fetch -q --all")
	if _, err := c.baseComponents.runLocalCommand(strings.Join(cmds, " && ")); err != nil {
		return err
	}
	targetBranch, err := c.resolveBranch(branch, gitDir)
	if err != nil {
		return err
	}
	cmds = []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(gitDir)))
	cmds = append(cmds, fmt.Sprintf("/usr/bin/env git checkout -q %s", common.ShellQuote(targetBranch)))
	cmds = append(cmds, fmt.Sprintf("/usr/bin/env git reset -q --hard %s", common.ShellQuote("origin/"+targetBranch)))
	cmd := strings.Join(cmds, " && ")
	_, err = c.baseComponents.runLocalCommand(cmd)
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
	// 空 filepath 引用出来是 '',那就是个空 pathspec,git 直接 fatal
	if filepath == "" {
		return nil, errors.New("文件路径为空")
	}
	// filepath 来自 git diff 的输出,仓库里完全可以存在带特殊字符的文件名,同样要引用
	destination := c.baseComponents.GetDeployFromDir()
	// 分支要放在 -- 前面:-- 之后的参数一律按 pathspec 解析,
	// 原先拼成 git log -- <branch> <filepath>,branch 被当成了一个匹配不到的路径,完全没生效
	gitLog := `/usr/bin/env git log `
	if branch != "" {
		gitLog += common.ShellQuote(branch) + " "
	}
	gitLog += `-- ` + common.ShellQuote(filepath) + ` | head -3 | tail -2`
	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("cd %s ", common.ShellQuote(destination)))
	cmds = append(cmds, gitLog)
	cmd := strings.Join(cmds, " && ")
	s, err := c.baseComponents.runLocalCommand(cmd)
	if err != nil {
		return nil, err
	} else {
		lines := strings.Split(s.Result, "\n")
		// pathspec 匹配不到提交时 git log 输出为空但退出码是 0,
		// 此时 lines 只有一个空串,直接取 lines[1] 会 panic
		if len(lines) < 2 {
			return nil, errors.New("取不到 " + filepath + " 的最后修改信息")
		}

		name := common.SubString(lines[0], 8, 100)
		time := common.SubString(lines[1], 8, 100)

		var fileinfo map[string]string
		fileinfo = make(map[string]string)
		fileinfo["name"] = name
		fileinfo["time"] = time
		return fileinfo, nil
	}
}
