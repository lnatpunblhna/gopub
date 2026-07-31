package components

import (
	"context"

	"github.com/bndr/gojenkins"
	"github.com/linclin/gopub/src/library/config"
	"github.com/linclin/gopub/src/library/logger"
	"net/url"
	"regexp"
	"strings"
)

type BasJenkins struct {
	baseComponents BaseComponents
}

func (c *BasJenkins) SetBaseComponents(b BaseComponents) {
	c.baseComponents = b
}

/**
 * 获取提交历史
 *
 */
type JenkinData struct {
	Build  string      `json:"build"`
	TarUrl string      `json:"tar_url"`
	MD5    interface{} `json:"md5"`
}

func (c *BasJenkins) GetCommitList(count int) ([]JenkinData, error) {
	//获取url 和job
	var list []JenkinData
	u, err := url.Parse(c.baseComponents.project.RepoUrl)
	jenkinsUrl := u.Scheme + "://" + u.Host
	jobs := strings.Split(u.Path, "/job/")
	job := strings.Trim(jobs[1], "/")
	jenkins := gojenkins.CreateJenkins(nil, jenkinsUrl, "")
	if config.String("JenkinsUserName") != "" {
		jenkins = gojenkins.CreateJenkins(nil, jenkinsUrl, config.String("JenkinsUserName"), config.String("JenkinsPwd"))
	}
	ctx := context.Background()
	_, err = jenkins.Init(ctx)
	if err != nil {
		logger.Error(err, "Jenkins Initialization failed")
		return list, err

	}
	builds, _ := jenkins.GetAllBuildIds(ctx, job)
	for _, b := range builds {
		build, _ := jenkins.GetBuild(ctx, job, b.Number)
		if len(build.Raw.Artifacts) == 0 {
			the_base := strings.Split(build.Base, "/")
			the_base_id := the_base[len(the_base)-1]
			var de_map JenkinData
			de_map.Build = the_base_id + "/null"
			de_map.TarUrl = "null"
			de_map.MD5 = ""
			list = append(list, de_map)
			continue
		}
		//取ID号
		path := build.Raw.Artifacts[0].RelativePath

		the_base := strings.Split(build.Base, "/")
		the_base_id := the_base[len(the_base)-1]
		reg := regexp.MustCompile("target/|-assembly.tar.gz|tar.gz")
		new_path := reg.ReplaceAllString(path, "")

		//new_path := strings.Replace(path, reg, "", -1)
		//拼接url
		url := "null"
		//var md5 interface{}

		url = jenkinsUrl + build.Base + "/artifact/" + path
		//md5 = build.Raw.MavenArtifacts
		var build_map JenkinData
		build_map.Build = the_base_id + "/" + new_path
		build_map.TarUrl = url
		//build_map.MD5 = md5
		list = append(list, build_map)
	}
	return list, nil
}
