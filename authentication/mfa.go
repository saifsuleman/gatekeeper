package authentication

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/saifsuleman/gatekeeper/logger"
	gomail "gopkg.in/mail.v2"
)

// multi-factor authentication
type MultiFactorAuth struct {
	ProxyAuthHandler ProxyAuthHandler  // instance of ProxyAuthHandler (Whitelist file storage handler)
	Emails           []string          // list of administrator email addresses
	AuthCodes        map[string]string // a map of authentication codes to the IP addresses they should whitelist
	DefaultApiUrl    string            // the API URL to encode in the links sent to the email
	ApiWhitelist     []string          // an IP address whitelist of the authenticated IP allowed to use administrator functions of the API
	Router           *mux.Router       // a reference to our HTTP router handler
	Logger           logger.Logger     // an instance of our custom logger
}

// constructor for our MFA instance
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

// checks whether or not the cryptographically secure code for an IP exists
func (mfa *MultiFactorAuth) DoesCodeExist(code string) bool {
	_, has := mfa.AuthCodes[code]
	return has
}

// uses the 'rand' library to generate a 256-bit secure base64 unique code
func (mfa *MultiFactorAuth) GenerateCode(ip string) (string, error) {
	// variable for our code to return later
	key := ""

	// while the key is empty or the key is already present in the map
	for key == "" || mfa.DoesCodeExist(key) {
		// creates a buffer of 32 bytes (32 * 8 = 256 bits)
		buf := make([]byte, 32)

		// uses rand to dump random bytes into the buffer and handles errors
		_, err := rand.Read(buf)
		if err != nil {
			return key, err
		}

		// encodes our random buffer into a base64 string and assign that to our 'key'
		key = base64.RawURLEncoding.EncodeToString(buf)
	}

	// updates the AuthCodes map to include the key as the key and ip as the value
	mfa.AuthCodes[key] = ip

	// returns the key (secure code) with a nil error (represents success)
	return key, nil
}

// function to get a code's IP and if it exists
func (mfa *MultiFactorAuth) GetCodeIP(code string) (string, bool) {
	// searches the map
	ip, has := mfa.AuthCodes[code]
	if !has {
		return "", false
	}
	return ip, true
}

// starts our HTTP server
func (mfa *MultiFactorAuth) Start(address string) {
	// declares HTTP routemap
	mfa.Router.HandleFunc("/api/authenticate", mfa.HandleAuthenticate)
	mfa.Router.HandleFunc("/api/log", mfa.wrapApiFunc("/api/log", mfa.ViewLog))

	// prints to the console window the address the API server is listening on
	fmt.Printf("Listening on: %s\n", address)

	// uses 'http' module to listen on the address with our router and handles error
	if err := http.ListenAndServe(address, mfa.Router); err != nil {
		panic(err)
	}
}

// returns whether or not a certain IP address has access to the API
func (mfa *MultiFactorAuth) HasApiAccess(r *http.Request) bool {
	// if the ApiWhitelist is empty, return true as all IPs are allowed
	if len(mfa.ApiWhitelist) == 0 {
		return true
	}

	// splits the address into its host and port and gets the address var
	// the port is unused so we call it _, we also handle errors by silently
	// returning false
	address, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	// for every value in the ApiWhitelist
	for _, v := range mfa.ApiWhitelist {
		// if the value is equal to the address, return true
		if v == address {
			return true
		}
	}
	// no values were found so we return false
	return false
}

// this function is a generator function so that only requests that are from
// an API-allowed IP address is able to be called - this utilises callbacks
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

// function to write the logger's cachedLog to a http response writer
// all return values of the logger is ignored, so we name them '_'
func (mfa *MultiFactorAuth) ViewLog(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, string(mfa.Logger.CachedLog))
}

// handler function for our /api/authenticate route
func (mfa *MultiFactorAuth) HandleAuthenticate(w http.ResponseWriter, r *http.Request) {
	// gets the "code" form value
	code := r.FormValue("code")
	// if the code is empty, we return an error
	if code == "" {
		_, _ = fmt.Fprint(w, "you must enter a code")
		return
	}
	// gets the IP address of a certain secure code
	// from the map and checks if its a valid code
	ip, valid := mfa.GetCodeIP(code)
	// if invalid, write an error back to the browser
	if !valid {
		_, _ = fmt.Fprint(w, "invalid code")
		return
	}
	// delete the auth code from the map as its now being processed
	// and we don't want to authenticate it twice
	delete(mfa.AuthCodes, code)

	// adds this IP address to the IP whitelist
	err := mfa.ProxyAuthHandler.AddWhitelistIP(ip)

	// calculates a response to send back to the browser
	// if no error from adding IP to whitelist, return "success",
	// else, return the error
	var response string
	if err == nil {
		response = "success"
	} else {
		response = fmt.Sprintf("error: %s", err)
	}

	// writes our calculated response back to browser
	_, _ = fmt.Fprint(w, response)
}

// function to check if an IP is authenticated and send an email
// alert if not
func (mfa *MultiFactorAuth) IsAuthenticated(ip string) bool {
	// uses the auth data handler and if whitelisted, return true
	if mfa.ProxyAuthHandler.IsWhitelisted(ip) {
		return true
	}

	// checks if ip has a code already generated
	for _, v := range mfa.AuthCodes {
		// if an IP has a code already generated
		// (and not authenticated yet), just return false
		if v == ip {
			return false
		}
	}

	// send the email alert in a subroutine and return false
	go mfa.SendEmailAlerts(ip)
	return false
}

// function to send email alerts to all the administrators
// for a connection attempt of a certain IP address
func (mfa *MultiFactorAuth) SendEmailAlerts(ip string) {
	// generates the secure unique code to identify
	// the IP in the email
	code, err := mfa.GenerateCode(ip)

	// if an error is returned, throw the error
	if err != nil {
		panic(err)
	}

	// uses the code to generate a link based off the DefaultApiUrl struct field
	// and the secure code
	link := fmt.Sprintf("%s/authenticate?code=%s", mfa.DefaultApiUrl, code)

	// gets the OS hostname to use in the email
	// and handles errors by throwing
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// generates the text body of the email alert
	body := fmt.Sprintf("RDP Login Attempt from %s.\nClick below to verify this IP.\n\n%s", ip, link)

	// uses the gomail library to generate a new SMTP dialer for the email
	// and configures TLS to work appropriately
	dialer := gomail.NewDialer("smtp.gmail.com", 587, "alerts@gatekeeper.io", "Password123")
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// array of emails to send all at once as a batch request to limit
	// network calls
	var messages []*gomail.Message

	// loops through all administrator email addresses
	for _, email := range mfa.Emails {
		// creates a new email message object and appends to the 'messages' array
		m := gomail.NewMessage()
		m.SetHeader("From", "RDP Gatekeeper <alerts@gatekeeper.io>")
		m.SetHeader("To", email)
		m.SetHeader("Subject", fmt.Sprintf("RDP Access Attempt on machine: %s", hostname))
		m.SetBody("text/plain", body)
		messages = append(messages, m)
	}

	// uses the diailer to send the array of emails in one
	// batch network call and handles errors
	if err := dialer.DialAndSend(messages...); err != nil {
		panic(err)
	}

	fmt.Printf("sent email to %v\n", mfa.Emails)
}
