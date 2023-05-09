package authentication

import (
	"encoding/json"
	"fmt"
	"os"
)

// IP whitelist handler for the proxdy
type ProxyAuthHandler struct {
	WhitelistFilepath string   // string field of the path to the whitelist file
	Whitelist         []string // list of IP addresses to represent the whitelist
}

// constructor for proxy auth handler - loads the file
// and decodes the existing whitelist as JSON and loads into array field
func NewProxyAuthHandler(filepath string) (ProxyAuthHandler, error) {
	// ProxyAuthHandler variable
	var handler ProxyAuthHandler

	// checks if file is present, if not:
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		// creates the file
		file, err := os.Create(filepath)
		// error handling
		if err != nil {
			return handler, fmt.Errorf("line 21: %s", err)
		}
		// attempts to write an empty JSON list and handles any errors
		if _, err := file.Write([]byte("[]")); err != nil {
			return handler, fmt.Errorf("line 24: %s", err)
		}
		// closes the file as we are now done with it
		file.Close()
	}
	// opens the file and handles errors
	file, err := os.Open(filepath)
	if err != nil {
		return handler, fmt.Errorf("line 29: %s", err)
	}

	// defers closing the file until after function ends
	defer file.Close()

	// local variable for whitelist to be loaded
	var whitelist []string

	// decode the JSON of the file into the whitelist ptr and handles errors
	if err := json.NewDecoder(file).Decode(&whitelist); err != nil {
		return handler, fmt.Errorf("line 36: %s", err)
	}

	// instantiates the proxy auth handler struct
	handler = ProxyAuthHandler{
		WhitelistFilepath: filepath,
		Whitelist:         whitelist,
	}

	// returns handler and no error to represent successful load
	return handler, nil
}

// function to save the contents of the Whitelist array to the file
func (p *ProxyAuthHandler) Save() error {
	// pointer to the whitelist file
	var file *os.File

	// if whitelist file does not exist
	if _, err := os.Stat(p.WhitelistFilepath); os.IsNotExist(err) {
		// create the file and assign it to our 'file' variable
		file, err = os.Create(p.WhitelistFilepath)
		// if an error is returned, return the error
		if err != nil {
			return err
		}
	} else {
		// if the file does exist, open it and assign the opened file to our 'file' var
		file, err = os.OpenFile(p.WhitelistFilepath, os.O_WRONLY, 0644)
		// if an error is returned, return the error
		if err != nil {
			return err
		}
	}

	// defers closing the file until after the function ends
	defer file.Close()

	// creates a new json encoder, encodes the Whitelist as a reference and returns the encoded string
	return json.NewEncoder(file).Encode(&p.Whitelist)
}

/**
**  Functions to add and remove IP addresses
**  from the whitelist.
**/
func (p *ProxyAuthHandler) AddWhitelistIP(ip string) error {
	// precondition check to ensure that the IP does not
	// already exist in this list.
	if p.GetWhitelistIPIndex(ip) > -1 {
		return fmt.Errorf("IP address already exists in the whitelist")
	}

	// appends to the whitelist
	p.Whitelist = append(p.Whitelist, ip)

	// saves the contents of whitelist to file
	// and returns the error if any
	return p.Save()
}

func (p *ProxyAuthHandler) RemoveWhitelistIP(ip string) error {
	// gets the index of this IP and returns error if not exists
	index := p.GetWhitelistIPIndex(ip)
	if index == -1 {
		return fmt.Errorf("IP address is not whitelisted")
	}
	// swaps the elements of this IP and last and then changes length of list
	last := len(p.Whitelist) - 1
	p.Whitelist[index], p.Whitelist[last] = p.Whitelist[last], p.Whitelist[index]
	p.Whitelist = p.Whitelist[:last]

	// saves the content of the modified whitelist to the file
	return p.Save()
}

// Function to find the index of an existing
// IP address if one exists, or returns -1
// to represent no indexes found
func (p *ProxyAuthHandler) GetWhitelistIPIndex(ip string) int {
	// loop through every element in the list
	for i, v := range p.Whitelist {
		// if the element we are at is equal to the IP, return index
		if v == ip {
			return i
		}
	}

	// return -1 to represent no indexes found
	return -1
}

func (p *ProxyAuthHandler) IsWhitelisted(ip string) bool {
	return p.GetWhitelistIPIndex(ip) > -1
}
