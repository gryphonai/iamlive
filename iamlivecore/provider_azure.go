package iamlivecore

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/oliveagle/jsonpath"
	"github.com/ucarion/urlpath"
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
func (azureProvider) PreRunSetup() {
	// Enable periodic terminal refresh when requested and install signal-based flush
	if *refreshRateFlag != 0 {
		setTerminalRefresh()
	}
	setINIConfigAndFileFlush()
}

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

func (azureProvider) GetPolicyDocument() []byte {
	actionsMap := make(map[string]bool)
	dataActionsMap := make(map[string]bool)

	for _, entry := range azureCallLog {
		for pathName, pathObj := range azureIamMap[strings.ToUpper(entry.HTTPMethod)] {
			pathmatch := urlpath.New(strings.ReplaceAll(strings.ReplaceAll(pathName, "{", ":"), "}", ""))
			pathmatchdata, ok := pathmatch.Match(entry.Path)
			if ok {
			PermissionLoop:
				for permissionName, permissionObj := range pathObj {
					if permissionObj.Condition.BodyPathExists != "" {
						var jsondata interface{}
						json.Unmarshal(entry.Body, &jsondata)
						_, err := jsonpath.JsonPathLookup(jsondata, permissionObj.Condition.BodyPathExists)
						if err != nil {
							continue PermissionLoop
						}
					}
					for pathName, pathValue := range permissionObj.Condition.PathEquals {
						if pathmatchdata.Params[pathName] != pathValue {
							continue PermissionLoop
						}
					}
					if permissionObj.IsDataAction {
						dataActionsMap[permissionName] = true
					} else {
						actionsMap[permissionName] = true
					}
				}
			}
		}
	}

	actionsList := make([]string, 0, len(actionsMap))
	for k := range actionsMap { actionsList = append(actionsList, k) }
	sort.Strings(actionsList)

	dataActionsList := make([]string, 0, len(dataActionsMap))
	for k := range dataActionsMap { dataActionsList = append(dataActionsList, k) }
	sort.Strings(dataActionsList)

	returnPolicy := AzureIAMPolicy{
		Actions:          actionsList,
		DataActions:      dataActionsList,
		NotDataActions:   make([]string, 0),
		AssignableScopes: make([]string, 0),
		IsCustom:         true,
	}

	doc, err := json.MarshalIndent(returnPolicy, "", "    ")
	if err != nil { panic(err) }
	return doc
}
