package configs

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"mirrors_status/internal/log"
)

type JobInfo struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	Token       string `yaml:"token"`
	Description string `yaml:"description"`
}

type JobInfoList []*JobInfo

type RepositoryInfo struct {
	Name string      `yaml:"name"`
	Jobs JobInfoList `yaml:"jobs"`
}
type RepositoryInfoList []*RepositoryInfo

type JenkinsConfig struct {
	PublishMirrors RepositoryInfoList `yaml:"publish_mirrors"`
	Repositories   RepositoryInfoList `yaml:"repositories"`
}


func NewJenkinsConfig() *JenkinsConfig {
	var jenkinsConfig JenkinsConfig
	log.Info("Loading jenkins configs")
	ymlFile, err := ioutil.ReadFile("configs/jenkins.yml")
	if err != nil {
		log.Errorf("Read YAML file found error:%#v", err)
	}

	err = yaml.Unmarshal(ymlFile, &jenkinsConfig)
	if err != nil {
		log.Errorf("Unmarshal YAML file found error:%#v", err)
	}

	return &jenkinsConfig
}