package authentication

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/saifsuleman/gatekeeper/logger"
	gomail "gopkg.in/mail.v2"
	"log"
	"net"
	"net/http"
	"os"
)

// multi-factor authentication
type MultiFactorAuth struct {
	ProxyAuthHandler ProxyAuthHandler
	Emails           []string
	AuthCodes        map[string]string
	DefaultApiUrl    string
	ApiWhitelist     []string
	Router           *mux.Router
	Logger           logger.Logger
}

func NewMFA(handler ProxyAuthHandler, logger logger.Logger, apiWhitelist []string, defaultApiUrl string, emails ...string) MultiFactorAuth {
	return MultiFactorAuth{
		ProxyAuthHandler: handler,
		Emails:           emails,
		AuthCodes:        map[string]string{},
		ApiWhitelist:     apiWhitelist,
		DefaultApiUrl:    defaultApiUrl,
		Logger:           logger,
		Router:           mux.NewRouter(),
	}
}

func (mfa *MultiFactorAuth) DoesCodeExist(code string) bool {
	_, has := mfa.AuthCodes[code]
	return has
}

// 256 bits
func (mfa *MultiFactorAuth) GenerateCode(ip string) (string, error) {
	key := ""
	for key == "" || mfa.DoesCodeExist(key) {
		buf := make([]byte, 32)

		_, err := rand.Read(buf)
		if err != nil {
			return key, err
		}

		key = base64.RawURLEncoding.EncodeToString(buf)
	}
	mfa.AuthCodes[key] = ip
	return key, nil
}

func (mfa *MultiFactorAuth) GetCodeIP(code string) (string, bool) {
	ip, has := mfa.AuthCodes[code]
	if !has {
		return "", false
	}
	return ip, true
}

func (mfa *MultiFactorAuth) Start(address string) {
	mfa.Router.HandleFunc("/api/authenticate", mfa.wrapApiFunc("/api/authenticate", mfa.HandleAuthenticate))
	mfa.Router.HandleFunc("/api/log", mfa.wrapApiFunc("/api/log", mfa.ViewLog))

	fmt.Printf("Listening on: %s\n", address)
	if err := http.ListenAndServe(address, mfa.Router); err != nil {
		panic(err)
	}
}

func (mfa *MultiFactorAuth) HasApiAccess(r *http.Request) bool {
	if len(mfa.ApiWhitelist) == 0 {
		return true
	}

	address, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	for _, v := range mfa.ApiWhitelist {
		if v == address {
			return true
		}
	}
	return false
}

func (mfa *MultiFactorAuth) wrapApiFunc(path string, f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if !mfa.HasApiAccess(r) {
			if address, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				log.Printf("Unauthorized API attempt [%s] from: %s\n", path, address)
			}
			_, _ = fmt.Fprint(w, "unauthorized")
			return
		}
		f(w, r)
	}
}

func (mfa *MultiFactorAuth) ViewLog(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, string(mfa.Logger.CachedLog))
}

func (mfa *MultiFactorAuth) HandleAuthenticate(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	if code == "" {
		_, _ = fmt.Fprint(w, "you must enter a code")
		return
	}
	ip, valid := mfa.GetCodeIP(code)
	if !valid {
		_, _ = fmt.Fprint(w, "invalid code")
		return
	}
	delete(mfa.AuthCodes, code)
	err := mfa.ProxyAuthHandler.AddWhitelistIP(ip)
	var response string
	if err == nil {
		response = "success"
	} else {
		response = fmt.Sprintf("error: %s", err)
	}
	_, _ = fmt.Fprint(w, response)
}

func (mfa *MultiFactorAuth) IsAuthenticated(ip string) bool {
	if mfa.ProxyAuthHandler.IsWhitelisted(ip) {
		return true
	}

	// checks if ip has a code already generated
	for _, v := range mfa.AuthCodes {
		if v == ip {
			return false
		}
	}

	go mfa.SendEmailAlerts(ip)
	return false
}

func (mfa *MultiFactorAuth) SendEmailAlerts(ip string) {
	code, err := mfa.GenerateCode(ip)
	if err != nil {
		panic(err)
	}
	link := fmt.Sprintf("%s/authenticate?code=%s", mfa.DefaultApiUrl, code)

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	body := fmt.Sprintf("RDP Login Attempt from %s.\nClick below to verify this IP.\n\n%s", ip, link)

	dialer := gomail.NewDialer("smtp.gmail.com", 587, "rdpgatekeeper@gmail.com", "Plasmetic12")
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	var messages []*gomail.Message

	for _, email := range mfa.Emails {
		m := gomail.NewMessage()
		m.SetHeader("From", "RDP Gatekeeper <rdpgatekeeper@gmail.com>")
		m.SetHeader("To", email)
		m.SetHeader("Subject", fmt.Sprintf("RDP Access Attempt on machine: %s", hostname))
		m.SetBody("text/plain", body)
		messages = append(messages, m)
	}

	if err := dialer.DialAndSend(messages...); err != nil {
		panic(err)
	}
}
