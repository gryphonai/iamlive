package iamlivecore

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
)

type azureProvider struct{}

func (azureProvider) Name() string      { return "azure" }
func (azureProvider) SupportsCSM() bool { return false }
func (azureProvider) LoadMaps() {
	debugln("Azure: Loading IAM map")
	if err := json.Unmarshal(bAzureIAMMap, &azureIamMap); err != nil {
		log.Fatal(err)
	}
}
func (azureProvider) PreRunSetup()      {}

// Azure does not require preloaded service files for current behavior
func (azureProvider) ReadServiceFiles() {}

func (azureProvider) HandleHTTPRequest(req *http.Request, _ string) (bool, []byte) {
	isAzureHostname, _ := regexp.MatchString(`^(?:management\.azure\.com)|(?:management\.core\.windows\.net)$`, req.Host)
	if !isAzureHostname {
		return false, nil
	}
	if *debugFlag {
		dumpReq(req)
	}
	body, _ := ioutil.ReadAll(req.Body)
	handleAzureRequest(req, body, 200)
	return true, body
}

func (azureProvider) RunCSM() {}

func (azureProvider) HostnamePattern() string {
	return "management\\.azure\\.com|management\\.core\\.windows\\.net"
}
