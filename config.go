package main

import "gopkg.in/yaml.v3"

var MoonConfig *Config

type Config struct {
	Endpoint string       `yaml:"endpoint"`
	Start    *StartConfig `yaml:"start"`
}

func init() {
	if file := getConfig(); file != nil {
		if err := yaml.NewDecoder(file).Decode(&MoonConfig); err != nil {
			logFatal(err)
		}
	}
}
