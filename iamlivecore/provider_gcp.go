package iamlivecore

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type gcpProvider struct{}

func (gcpProvider) Name() string      { return "gcp" }
func (gcpProvider) SupportsCSM() bool { return false }
func (gcpProvider) LoadMaps() {
	if err := json.Unmarshal(bGCPIAMMap, &gcpIamMap); err != nil {
		log.Fatal(err)
	}
}
func (gcpProvider) PreRunSetup()      {}

func (gcpProvider) ReadServiceFiles() {
	// Prefer live Discovery API; fallback to embedded JSONs on failure
	if !loadGCPFromDiscovery() {
		file, err := gcpServiceFiles.Open("google-api-go-client/api-list.json")
		if err != nil {
			panic(err)
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		var apiList GCPAPIListFile
		if json.Unmarshal(data, &apiList) != nil {
			panic(err)
		}
		for _, apiItem := range apiList.Items {
			version := strings.ToLower(strings.ReplaceAll(apiItem.Version, "_", "/"))
			if version == "alpha" {
				version = "v0.alpha"
			} else if version == "beta" {
				version = "v0.beta"
			}
			file, err := gcpServiceFiles.Open("google-api-go-client/" + strings.ToLower(apiItem.Name) + "/" + version + "/" + strings.ToLower(apiItem.Name) + "-api.json")
			if err != nil {
				file, err = gcpServiceFiles.Open("google-api-go-client/" + strings.ToLower(apiItem.Name) + "/" + strings.ToLower(apiItem.Version) + "/" + strings.ToLower(apiItem.Name) + "-api.json")
				if err != nil {
					panic(err)
				}
			}
			data, err := ioutil.ReadAll(file)
			if err != nil {
				panic(err)
			}
			var def GCPServiceDefinition
			if json.Unmarshal(data, &def) != nil {
				panic("bad json")
			}
			u, _ := url.Parse(def.RootURL)
			def.RootDomain = u.Hostname()
			gcpServiceDefinitions = append(gcpServiceDefinitions, def)
		}
	}
}

func (gcpProvider) HandleHTTPRequest(req *http.Request, _ string) (bool, []byte) {
	isGCPHostname, _ := regexp.MatchString(`^.*\\.googleapis\\.com$`, req.Host)
	if !isGCPHostname {
		return false, nil
	}
	if *debugFlag {
		dumpReq(req)
	}
	body, _ := ioutil.ReadAll(req.Body)
	handleGCPRequest(req, body, 200)
	return true, body
}

func (gcpProvider) RunCSM() {}

func (gcpProvider) HostnamePattern() string {
	return ".*\\.googleapis\\.com"
}

// loadGCPFromDiscovery fetches Google Cloud APIs from the Discovery API and populates gcpServiceDefinitions.
// Returns true if at least one service definition was successfully loaded.
func loadGCPFromDiscovery() bool {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://www.googleapis.com/discovery/v1/apis?preferred=true")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}

	var apiList GCPAPIListFile
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&apiList); err != nil {
		return false
	}
	if len(apiList.Items) == 0 {
		return false
	}

	for _, item := range apiList.Items {
		if item.DiscoveryRestURL == "" {
			// older fields sometimes use discoveryRestUrl; if not present, skip
			continue
		}
		r2, err := client.Get(item.DiscoveryRestURL)
		if err != nil {
			continue
		}
		data, err := ioutil.ReadAll(r2.Body)
		r2.Body.Close()
		if err != nil {
			continue
		}

		var def GCPServiceDefinition
		if json.Unmarshal(data, &def) != nil {
			continue
		}
		u, err := url.Parse(def.RootURL)
		if err == nil {
			def.RootDomain = u.Hostname()
		}
		gcpServiceDefinitions = append(gcpServiceDefinitions, def)
	}

	return len(gcpServiceDefinitions) > 0
}
