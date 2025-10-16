package iamlivecore

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
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
func (gcpProvider) PreRunSetup() {
	// Set up periodic terminal refresh if configured, and signal-based
	// output flushing (same mechanism used by AWS). For non-AWS providers,
	// setINIConfigAndFileFlush will only modify AWS config if --set-ini is
	// explicitly provided, so it is safe to call here to install the signal
	// handler.
	if *refreshRateFlag != 0 {
		setTerminalRefresh()
	}
	setINIConfigAndFileFlush()
}

func (gcpProvider) ReadServiceFiles() {
	debugln("GCP: Loading service definitions (Discovery API preferred)")
	// Prefer live Discovery API; fallback to embedded JSONs on failure
	if !loadGCPFromDiscovery() {
		debugln("GCP: Discovery API unavailable or returned no items; using embedded JSONs")
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
	debugf("GCP: loaded %d service definitions", len(gcpServiceDefinitions))
}

func (gcpProvider) HandleHTTPRequest(req *http.Request, _ string) (bool, []byte) {
	isGCPHostname, _ := regexp.MatchString(`^.*\.googleapis\.com$`, req.Host)
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
	debugln("GCP: Fetching API list from Discovery API")
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://www.googleapis.com/discovery/v1/apis?preferred=true")
	if err != nil {
		debugf("GCP Discovery: failed to fetch API list: %v", err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		debugf("GCP Discovery: non-200 fetching API list: %d", resp.StatusCode)
		return false
	}

	var apiList GCPAPIListFile
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&apiList); err != nil {
		debugf("GCP Discovery: failed to decode API list: %v", err)
		return false
	}
	if len(apiList.Items) == 0 {
		debugln("GCP Discovery: API list returned 0 items")
		return false
	}

	// Determine worker count
	workers := 10
	if gcpDiscoveryParallelFlag != nil && *gcpDiscoveryParallelFlag > 0 {
		workers = *gcpDiscoveryParallelFlag
	}
	debugf("GCP Discovery: starting parallel fetch with %d workers", workers)

	jobs := make(chan GCPAPIListItem)
	results := make(chan GCPServiceDefinition, 32)
	var wg sync.WaitGroup

	// Workers
	workerFn := func(id int) {
		defer wg.Done()
		for item := range jobs {
			if item.DiscoveryRestURL == "" {
				debugf("GCP Discovery: skipping %s:%s (no discoveryRestUrl)", item.Name, item.Version)
				continue
			}
			debugf("GCP Discovery: [w%d] fetching %s:%s (%s)", id, item.Name, item.Version, item.DiscoveryRestURL)
			r2, err := client.Get(item.DiscoveryRestURL)
			if err != nil {
				debugf("GCP Discovery: failed to fetch %s:%s from %s: %v", item.Name, item.Version, item.DiscoveryRestURL, err)
				continue
			}
			data, err := ioutil.ReadAll(r2.Body)
			r2.Body.Close()
			if err != nil {
				debugf("GCP Discovery: failed reading body for %s:%s: %v", item.Name, item.Version, err)
				continue
			}

			var def GCPServiceDefinition
			if json.Unmarshal(data, &def) != nil {
				debugf("GCP Discovery: failed to parse discovery document for %s:%s", item.Name, item.Version)
				continue
			}
			u, err := url.Parse(def.RootURL)
			if err == nil {
				def.RootDomain = u.Hostname()
			}
			debugf("GCP Discovery: loaded %s:%s root=%s domain=%s", item.Name, item.Version, def.RootURL, def.RootDomain)
			results <- def
		}
	}

	// Start workers
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go workerFn(i + 1)
	}

	// Enqueue jobs
	go func() {
		for _, item := range apiList.Items {
			if item.DiscoveryRestURL == "" {
				debugf("GCP Discovery: skipping %s:%s (no discoveryRestUrl)", item.Name, item.Version)
				continue
			}
			jobs <- item
		}
		close(jobs)
	}()

	// Close results when done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	loaded := 0
	local := make([]GCPServiceDefinition, 0, 256)
	for def := range results {
		local = append(local, def)
		loaded++
	}

	if loaded > 0 {
		gcpServiceDefinitions = append(gcpServiceDefinitions, local...)
	}
	debugf("GCP Discovery: parallel load complete, %d services loaded", loaded)
	return loaded > 0
}
