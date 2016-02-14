package main

import (
	"bytes"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
)

type config struct {
	ProxyList string `yaml:"proxyList"`
	Check     struct {
		URL      string `yaml:"URL"`
		String   string `yaml:"string"`
		Interval string `yaml:"interval"`
		Timeout  string `yaml:"timeout"`
	}
	Bind         string `yaml:"bind"`
	WorkersCount int    `yaml:"workersCount"`
	MaxTry       int    `yaml:"maxTry"`
	Debug        bool   `yaml:"debug"`
}

var (
	cfg config
)

const (
	appName    string = "uproxy"
	appVersion string = "0.0.1"
)

func readConfig() error {

	viper.SetConfigType("yaml")
	viper.SetDefault("proxyList", "/etc/proxy.list")
	viper.SetDefault("check", map[string]interface{}{
		"url":      "http://ya.ru",
		"string":   "yandex",
		"interval": "60m",
		"timeout":  "5s",
	})
	viper.SetDefault("bind", "0.0.0.0:8080")
	viper.SetDefault("workersCount", 20)
	viper.SetDefault("maxTry", 3)
	viper.SetDefault("debug", false)

	var configFile = flag.StringP("config", "c", "/etc/"+appName+".yaml",
		"full path to config")
	var showVersion = flag.BoolP("version", "v", false, "version")
	flag.Parse()

	if *showVersion {
		log.Println(appVersion)
		os.Exit(0)
	}

	file, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}

	err = viper.ReadConfig(bytes.NewReader(file))
	if err != nil {
		return err
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return err
	}
	return nil
}
