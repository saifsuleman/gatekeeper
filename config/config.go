package config

import (
	_ "embed"
	"encoding/json"
	"io/ioutil"
	"os"
)

//go:embed config.json
var defaultConfig string

type ApplicationConfig struct {
	ProxyAddress    string   `json:"proxyAddress"`
	RedirectAddress string   `json:"redirectAddress"`
	ApiAddress      string   `json:"apiAddress"`
	LoggerPath      string   `json:"loggerPath"`
	DefaultApiUrl   string   `json:"defaultApiUrl"`
	ApiWhitelist    []string `json:"apiWhitelist"`
	Emails          []string `json:"emails"`
}

func IsTextValidConfig(text string) error {
	var config ApplicationConfig
	if err := json.Unmarshal([]byte(text), &config); err != nil {
		return err
	}
	return nil
}

func NewApplicationConfig(filepath string) ApplicationConfig {
	if err := IsTextValidConfig(defaultConfig); err != nil {
		panic(err)
	}

	var config ApplicationConfig
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		err := ioutil.WriteFile(filepath, []byte(defaultConfig), 0644)
		if err != nil {
			panic(err)
		}
	}

	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		panic(err)
	}

	return config
}
