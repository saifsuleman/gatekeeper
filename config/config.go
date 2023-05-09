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
	ProxyAddress    string   `json:"proxyAddress"`    // the address the tcp proxy server is listening on
	RedirectAddress string   `json:"redirectAddress"` // the address the tcp proxy server will use as its target service to piping
	ApiAddress      string   `json:"apiAddress"`      // the address the REST API will be listening on
	LoggerPath      string   `json:"loggerPath"`      // the path to the output file of the program's log
	DefaultApiUrl   string   `json:"defaultApiUrl"`   // the publicly accessible link to the REST API to be used in embedded in the email links
	ApiWhitelist    []string `json:"apiWhitelist"`    // the IP address whitelist to access sensitive information from the REST API such as the log
	Emails          []string `json:"emails"`          // the list of administrator email addresses that the program should email alerts to
}

// determines whether or not a text string is a
// valid configuration JSON
func IsTextValidConfig(text string) error {
	// variable for our config
	var config ApplicationConfig
	// attempts to decode the text into our config variable,
	// then return the error
	return json.Unmarshal([]byte(text), &config)
}

// constructor for our ApplicationConfig
func NewApplicationConfig(filepath string) (ApplicationConfig, error) {
	// variable for our new ApplicationConfig
	var config ApplicationConfig

	// first checks if our defaultConfig is valid to prevent
	// errors that occurred from compilation and make them obvious
	if err := IsTextValidConfig(defaultConfig); err != nil {
		return config, err
	}

	// checks if the file exists, if it doesn't, then paste in
	// the default configuration
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		err := ioutil.WriteFile(filepath, []byte(defaultConfig), 0644)
		if err != nil {
			return config, err
		}
	}

	// opens a connection to the file path, if an error is
	// present, then throw the error
	file, err := os.Open(filepath)
	if err != nil {
		return config, err
	}

	// decodes the file contents into the config using
	// a pointer, and if an error is returned, then
	// throw the error
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}

	// returns our newly defined config
	return config, nil
}
