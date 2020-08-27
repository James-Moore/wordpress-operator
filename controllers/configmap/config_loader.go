package configmap

import (
	"github.com/ghodss/yaml"
	"io/ioutil"
)

type Config struct {
	Message string `yaml:"message"`
}

func (configMap *Config) LoadConfig(configFile string) (err error) {
	err = nil

	configData, err := ioutil.ReadFile(configFile)
	err = yaml.Unmarshal(configData, configMap)
	return err
}
