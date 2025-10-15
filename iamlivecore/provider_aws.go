package iamlivecore

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

type awsProvider struct{}

func (awsProvider) Name() string      { return "aws" }
func (awsProvider) SupportsCSM() bool { return true }
func (awsProvider) LoadMaps() {
	debugln("AWS: Loading IAM maps and SAR definitions")
	// Load AWS IAM maps and definitions
	if *overrideAwsMapFlag != "" {
		b, err := os.ReadFile(*overrideAwsMapFlag)
		if err != nil {
			log.Fatal(err)
		}
		if err := json.Unmarshal(b, &iamMap); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := json.Unmarshal(bIAMMap, &iamMap); err != nil {
			log.Fatal(err)
		}
	}
	if err := json.Unmarshal(bIAMSAR, &iamDef); err != nil {
		panic(err)
	}
}

// PreRunSetup performs AWS-specific setup like terminal refresh and INI configuration
func (awsProvider) PreRunSetup() {
	if *refreshRateFlag != 0 {
		setTerminalRefresh()
	}
	setINIConfigAndFileFlush()
}

// ReadServiceFiles loads the embedded AWS API definitions
func (awsProvider) ReadServiceFiles() {
	serviceDirs, err := serviceFiles.ReadDir("apis")
	if err != nil {
		panic(err)
	}

	for _, serviceEntry := range serviceDirs {
		versionDirs, err := serviceFiles.ReadDir("apis/" + serviceEntry.Name())
		if err != nil {
			panic(err)
		}

		latestDir := ""
		for _, versionEntry := range versionDirs {
			if latestDir == "" || versionEntry.Name() > latestDir {
				latestDir = versionEntry.Name()
			}
		}

		file, err := serviceFiles.Open("apis/" + serviceEntry.Name() + "/" + latestDir + "/api-2.json")
		if err != nil {
			panic(err)
		}

		data, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}

		var def ServiceDefinition
		if json.Unmarshal(data, &def) != nil {
			panic(err)
		}

		serviceDefinitions = append(serviceDefinitions, def)
	}
	debugf("AWS: loaded %d service definitions", len(serviceDefinitions))
}

// HandleHTTPRequest processes AWS requests in proxy mode
func (awsProvider) HandleHTTPRequest(req *http.Request, awsRedirectHost string) (bool, []byte) {
	isAWSHostname, _ := regexp.MatchString(`^.*\\.amazonaws\\.com(?:\\.cn)?$`, req.Host)
	if !isAWSHostname {
		return false, nil
	}
	if *debugFlag {
		dumpReq(req)
	}
	body, _ := ioutil.ReadAll(req.Body)
	handleAWSRequest(req, body, 200)
	if awsRedirectHost != "" {
		req.URL.Host = awsRedirectHost
		req.Host = awsRedirectHost
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return true, body
}

func (awsProvider) RunCSM() {
	listenForEvents()
	handleLoggedCall()
}

func (awsProvider) HostnamePattern() string {
	return ".*\\.amazonaws\\.com(?:\\.cn)?"
}
