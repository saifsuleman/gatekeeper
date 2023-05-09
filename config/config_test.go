package config

import (
	"testing"
)

func TestConfigurationSystem(t *testing.T) {
	config, err := NewApplicationConfig("testing_configuration.json")

	if err != nil {
		t.Error(err)
	}

	t.Logf(
		`
		LOADED TEST CONFIGURATION:
		ProxyAddress: %s,
		RedirectAddress: %s,
		ApiAddress: %s,
		LoggerPath: %s,
		DefaultApiUrl: %s,
		ApiWhitelist: %v,
		Emails: %v
	`, config.ProxyAddress, config.RedirectAddress, config.ApiAddress, config.LoggerPath, config.DefaultApiUrl, config.ApiWhitelist, config.Emails)
}
