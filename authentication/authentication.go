package authentication

import (
	"encoding/json"
	"fmt"
	"os"
)

type ProxyAuthHandler struct {
	WhitelistFilepath string
	Whitelist         []string
}

func NewProxyAuthHandler(filepath string) (ProxyAuthHandler, error) {
	var handler ProxyAuthHandler

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		file, err := os.Create(filepath)
		if err != nil {
			return handler, fmt.Errorf("line 21: %s", err)
		}
		if _, err := file.Write([]byte("[]")); err != nil {
			return handler, fmt.Errorf("line 24: %s", err)
		}
		file.Close()
	}
	file, err := os.Open(filepath)
	if err != nil {
		return handler, fmt.Errorf("line 29: %s", err)
	}

	defer file.Close()
	var whitelist []string
	if err := json.NewDecoder(file).Decode(&whitelist); err != nil {
		return handler, fmt.Errorf("line 36: %s", err)
	}

	handler = ProxyAuthHandler{
		WhitelistFilepath: filepath,
		Whitelist:         whitelist,
	}

	return handler, nil
}

func (p *ProxyAuthHandler) Save() error {
	var file *os.File
	if _, err := os.Stat(p.WhitelistFilepath); os.IsNotExist(err) {
		file, err = os.Create(p.WhitelistFilepath)
		if err != nil {
			return err
		}
	} else {
		file, err = os.OpenFile(p.WhitelistFilepath, os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(&p.Whitelist)
}

func (p *ProxyAuthHandler) AddWhitelistIP(ip string) error {
	index := p.GetWhitelistIPIndex(ip)
	if index > -1 {
		return fmt.Errorf("ip already whitelisted")
	}
	p.Whitelist = append(p.Whitelist, ip)
	return p.Save()
}

func (p *ProxyAuthHandler) RemoveWhitelistIP(ip string) error {
	index := p.GetWhitelistIPIndex(ip)
	if index < 0 {
		return fmt.Errorf("ip not whitelisted")
	}
	last := len(p.Whitelist) - 1
	p.Whitelist[index], p.Whitelist[last] = p.Whitelist[last], p.Whitelist[index]
	p.Whitelist = p.Whitelist[:last]
	return p.Save()
}

func (p *ProxyAuthHandler) GetWhitelistIPIndex(ip string) int {
	for i, v := range p.Whitelist {
		if v == ip {
			return i
		}
	}
	return -1
}

func (p *ProxyAuthHandler) IsWhitelisted(ip string) bool {
	return p.GetWhitelistIPIndex(ip) > -1
}
