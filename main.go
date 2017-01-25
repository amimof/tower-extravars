package main

import (
	"net/http"
	"crypto/tls"
	"encoding/json"
	"encoding/gob"
    "bytes"
	"io/ioutil"
	"flag"
	"net/url"
	"path"
	"encoding/base64"
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	"strings"
	"reflect"
	"github.com/amimof/loglevel-go"
)

var (
	jobtemplateid string
	verbosity int
	towerurl string
	username string
	password string
	inputfile string
	strategy string
	insecure bool 
	confirm bool
)

func encodeCredentials(user, pass string) string {
	credentials := []byte(fmt.Sprintf("%s:%s", user, pass))
	return base64.StdEncoding.EncodeToString(credentials)
}

func onRedirect(req *http.Request, via []*http.Request) error {
	// Re-set the auth header
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodeCredentials(username, password)))
	return nil
}

// Returns true if path exists
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// Return true if path is a file
func isFile(path string) bool {
	file, err := os.Stat(path)
	if err == nil && file.IsDir() != true {
		return true
	}
	return false
}

func isStrategy(str string) bool {
	strategies := []string{"append", "update", "replace", "delete"}
	for _, s := range strategies {
		if s == str {
			return true
		}
	}
	return false
}

// Checks if the key str exists in list
func keyExists(list map[string]interface{}, key string) bool {
	for k, _ := range list {
		if k == key {
			return true
		}
	}
	return false
}

type JobTemplate struct {
	Id string `json"id"`
	Name string `json"name"`
	Desription string `json"description"`
	Url string `json"url"`
	ExtraVars map[string]interface{} `json"extra_vars"`
}

func GetBytes(key interface{}) ([]byte, error) {
    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    err := enc.Encode(key)
    if err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}

func main() {

	// Read arguments
	flag.StringVar(&jobtemplateid, "i", "", "Job Template ID separated by comma.")
	flag.StringVar(&towerurl, "h", "https://localhost/", "URL to Ansible Tower.")
	flag.StringVar(&username, "u", "", "Username to Ansible Tower.")
	flag.StringVar(&password, "p", "", "Password to Ansible Tower.")
	flag.BoolVar(&insecure, "k", false, "Allow connection to Ansible Tower without a valid certificate.")
	flag.StringVar(&inputfile, "f", "", "Path to input yaml file. The program reads its content and applies it to the remote Ansible Tower server using one of the update strategies.")
	flag.StringVar(&strategy, "s", "update", "Update strategy to use when updating extra vars with input file. Choices are: APPEND - Add missing fields and their values. REPLACE - Replace extra_vars in Ansible Tower with content of the file file. UPDATE - Replace extra_vars values in Ansible Tower with content of the file file, if the field exists in both places. DELETE - Delete all fields that are defined in the file file from extra_vars in Ansible Tower.")
	flag.IntVar(&verbosity, "v", 1, "Verbosity. ERROR=0, WARN=1, INFO=2, DEBUG=3.")
	flag.BoolVar(&confirm, "c", true, "Confirm changes made and perform patch operation. Set to false if you want to review changes and not implement them.")
	flag.Parse()

	// Setup logging
	log := *loglevel.New().SetLevel(verbosity)
	log.PrintTime = false

	log.Debugf("Confirm changes is '%t'", confirm)

	// Make strategy lowercase 
	strategy = strings.ToLower(strategy)
	log.Debugf("Strategy is '%s'", strategy)

	// Check if strategy is correct
	if !isStrategy(strategy) {
		log.Error("Strategy must be one of append, update, replace.")
		flag.PrintDefaults()
	}

	log.Debugf("Input file is '%s'", inputfile)
	
	// Check so that user gave us an input file
	if inputfile == "" {
		log.Errorf("Flag not defined: -f")
	}

	// Check so that file file exists
	if !exists(inputfile) {
		log.Errorf("File '%s' not found.", inputfile)
	}

	// Check so that file file is a file
	if !isFile(inputfile) {
		log.Errorf("File '%s' is not a file.", inputfile)
	}

	// Parse the url provided on the command line
	apiurl, err := url.Parse(towerurl)
	if err != nil {
		log.Error(err)
	}

	// Check so the id argument is ok
	if jobtemplateid == "" {
		log.Error("At least one job template id must be provided and be greater than 0.")
	}

	// Make an array of the ids
	ids := strings.Split(jobtemplateid, ",")

	// Create the transport
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecure,		
		},
	}	
	// Create the http client
	client := &http.Client{
		Transport: tr,
    	CheckRedirect: onRedirect,
	}

	for i, id := range ids { 

		id = strings.TrimSpace(id)

		log.Debugf("At job template %s (%d/%d)", id, int(i)+1, len(ids))

		// Build the path to job templates
		apiurl.Path = path.Join(apiurl.Path, "/api/v1/job_templates/", id, "/")
		apiurl.Path = apiurl.Path+"/"
		log.Debugf("Skip Verify SSL Certs: %t", insecure)

		// Create the request 
		req, err := http.NewRequest("GET", apiurl.String(), nil)
		if err != nil {
			log.Error(err)
		}

		// Add the authorization header and execute the request
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodeCredentials(username, password)))
		resp, err := client.Do(req)
		log.Debugf("%s %s", req.Method, req.URL)
		if err != nil {
			log.Error(err)
		}

		// Log the response headers 
		log.Debugf("%s, %d (%s)", resp.Proto, resp.StatusCode, resp.Status)

		// Read the body of the response
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error(err)
		}

		// Unmarshal the http response body into a map array
		var bo map[string]interface{}
		json.Unmarshal(b, &bo)

		// The server might respond with an unexpected response. 
		// This will cause bellow type assertion to fail since the key extra_vars doesn't exist in the response body
		if !keyExists(bo, "extra_vars")	{
			log.Errorf("Unexpected response from server")
		}

		// Read the extra vars field of the json response
		e, err := ioutil.ReadAll(strings.NewReader(bo["extra_vars"].(string)))
		if err != nil {
			log.Error(err)
		}

		// Yaml unmarshal the extra vars field
		var eo map[string]interface{}
		yaml.Unmarshal(e, &eo)

		log.Debugf("Unmodified extra_vars: (len: %d)", len(eo))
		for k, v := range eo {
			log.Debugf("\t%s: %s", k, v)
		}

		// Read the input file provided by the user
		s, err := ioutil.ReadFile(inputfile)
		if err != nil {
			log.Error(err)
		}

		// Yaml unmashal the input file yaml content
		var so map[string]interface{}
		yaml.Unmarshal(s, &so)

		/* Scenarios
		Does not support recursivenes
		1. APPEND - Add missing fields and their values
		2. REPLACE - Replace entire source with input file
		3. UPDATE - Replace partial source with input file
		*/

		log.Debugf("Patching extra_vars with:")

		// This is where the magic happens
		switch strategy {
			case "append":
				for k, v := range so {
					if !keyExists(eo, k) {
						eo[k] = v
						log.Debugf("[%s] %s : %s", "A", k, v)
					}
				}
			case "update":
				for k, v := range so {
					if keyExists(eo, k) {

						// This if-clause is just to figure out if a value has been changed and if so, display it for the user.
						// Unfortunately, this is complicated with map arrays so we just replace the values if keys exists in both places.
						if reflect.TypeOf(eo[k]).Kind() != reflect.Slice {

							a, err := GetBytes(eo[k])
							if err != nil {
								log.Error(err)
							}

							b, err := GetBytes(so[k])
							if err != nil {
								log.Error(err)
							}

							if bytes.Compare(a, b) < 0 {
								log.Debugf("[%s] %s : %s", "U", k, v)	
							}
						}
						eo[k] = v
					}
				}
			case "replace":
				for k, v := range eo {
					if !keyExists(so, k) {
						log.Debugf("[%s] %s : %s", "-", k, v)	
					} else {
						log.Debugf("[%s] %s : %s", "R", k, v)
					}
				}
				for k, v := range so {
					if !keyExists(eo, k) {
						log.Debugf("[%s] %s : %s", "+", k, v)
					} 
				}
				eo = so
			case "delete":
				for k, v := range so {
					if keyExists(eo, k) {
						delete(eo, k)
						log.Debugf("[%s] %s : %s", "-", k, v)
					}
				}
		}

		log.Debugf("Patched length: %d", len(eo))

		// Build the extra_vars field that we send back to the PATCH request
		y, err := yaml.Marshal(eo)
		if err != nil {
			log.Error(err)
		}
		ss := fmt.Sprintf("{\"extra_vars\": \"%s\"}", y)

		// Replace new-lines with escaped new-lines so that tower api doesnt freak out
		ss = strings.Replace(ss, "\n", "\\n", -1)

		// Create the byte which we send to the server
		sss := []byte(ss)

		// Create the PATCH request 
		req, err = http.NewRequest("PATCH", apiurl.String(), bytes.NewBuffer([]byte(sss)))
		if err != nil {
			log.Error(err)
		}

		// Add the authorization header and execute the request
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodeCredentials(username, password)))
		req.Header.Add("Content-Type", "application/json")
		log.Debugf("Request Body: %q", sss)
		log.Debugf("%s %s (len: %d)", req.Method, req.URL, len(sss))
		
		if confirm { 
			resp, err = client.Do(req)
			if err != nil {
				log.Error(err)
			}

			// Log the response headers 
			log.Debugf("%s, %d (%s)", resp.Proto, resp.StatusCode, resp.Status)
			b, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Error(err)
			}
		} else {
			log.Warnf("Confirm is %t. No changes are beeing made.", confirm)
		}
	}	

}
