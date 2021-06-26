package authentication

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"net/smtp"
)

// multi-factor authentication

type MultiFactorAuth struct {
	ProxyAuthHandler ProxyAuthHandler
	Emails           []string
	AuthCodes        map[string]string
	Router           *mux.Router
}

func NewMFA(handler ProxyAuthHandler, emails ...string) MultiFactorAuth {
	return MultiFactorAuth{
		ProxyAuthHandler: handler,
		Emails:           emails,
		AuthCodes:        map[string]string{},
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
	mfa.Router.HandleFunc("/api/authenticate", mfa.HandleAuthenticate)
	fmt.Printf("Listening on: %s\n", address)
	if err := http.ListenAndServe(address, mfa.Router); err != nil {
		panic(err)
	}
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
	from := "rdpgateway@gmail.com"
	password := "Plasmetic12"

	code, err := mfa.GenerateCode(ip)
	if err != nil {
		panic(err)
	}
	link := fmt.Sprintf("http://rdp.plasmoid.io/api/authenticate?code=%s", code)

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	message := []byte(fmt.Sprintf("RDP Login Attempt from %s. Click below to verify this IP.\n\n%s", ip, link))
	auth := smtp.PlainAuth("", from, password, smtpHost)

	if err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, mfa.Emails, message); err != nil {
		panic(err)
	}
	fmt.Println("Emails sent!")
}
