package iamlivecore

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime/pprof"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/ini.v1"
)

// CLI args
var providerFlag *string
var setiniFlag *bool
var profileFlag *string
var failsonlyFlag *bool
var outputFileFlag *string
var refreshRateFlag *int
var sortAlphabeticalFlag *bool
var hostFlag *string
var modeFlag *string
var bindAddrFlag *string
var caBundleFlag *string
var caKeyFlag *string
var accountIDFlag *string
var backgroundFlag *bool
var overrideAwsMapFlag *string
var debugFlag *bool
var forceWildcardResourceFlag *bool
var cpuProfileFlag = flag.String("cpu-profile", "", "write a CPU profile to this file (for performance testing purposes)")
var csmPortFlag *int
var awsRedirectHostFlag *string
var gcpDiscoveryParallelFlag *int

func parseConfig() {
	provider := "aws"
	setIni := false
	profile := "default"
	failsOnly := false
	outputFile := ""
	refreshRate := 0
	sortAlphabetical := false
	host := "127.0.0.1"
	mode := "csm"
	bindAddr := "127.0.0.1:10080"
	caBundle := "~/.iamlive/ca.pem"
	caKey := "~/.iamlive/ca.key"
	accountID := ""
	background := false
	overrideAwsMap := ""
	debug := false
	forceWildcardResource := false
	csmPort := 31000
	awsRedirectHost := ""
	gcpDiscoveryParallel := 10

	cfgfile, err := homedir.Expand("~/.iamlive/config")
	if err == nil {
		cfg, err := ini.Load(cfgfile)
		if err == nil {
			if cfg.Section("").HasKey("provider") {
				provider = cfg.Section("").Key("provider").String()
			}
			if cfg.Section("").HasKey("set-ini") {
				setIni, _ = cfg.Section("").Key("set-ini").Bool()
			}
			if cfg.Section("").HasKey("profile") {
				profile = cfg.Section("").Key("profile").String()
			}
			if cfg.Section("").HasKey("fails-only") {
				failsOnly, _ = cfg.Section("").Key("fails-only").Bool()
			}
			if cfg.Section("").HasKey("output-file") {
				outputFile = cfg.Section("").Key("output-file").String()
			}
			if cfg.Section("").HasKey("refresh-rate") {
				refreshRate, _ = cfg.Section("").Key("refresh-rate").Int()
			}
			if cfg.Section("").HasKey("sort-alphabetical") {
				sortAlphabetical, _ = cfg.Section("").Key("sort-alphabetical").Bool()
			}
			if cfg.Section("").HasKey("host") {
				host = cfg.Section("").Key("host").String()
			}
			if cfg.Section("").HasKey("mode") {
				mode = cfg.Section("").Key("mode").String()
			}
			if cfg.Section("").HasKey("bind-addr") {
				bindAddr = cfg.Section("").Key("bind-addr").String()
			}
			if cfg.Section("").HasKey("ca-bundle") {
				caBundle = cfg.Section("").Key("ca-bundle").String()
			}
			if cfg.Section("").HasKey("ca-key") {
				caKey = cfg.Section("").Key("ca-key").String()
			}
			if cfg.Section("").HasKey("account-id") {
				accountID = cfg.Section("").Key("account-id").String()
			}
			if cfg.Section("").HasKey("background") {
				background, _ = cfg.Section("").Key("background").Bool()
			}
			if cfg.Section("").HasKey("override-aws-map") {
				overrideAwsMap = cfg.Section("").Key("override-aws-map").String()
			}
			if cfg.Section("").HasKey("debug") {
				debug, _ = cfg.Section("").Key("debug").Bool()
			}
			if cfg.Section("").HasKey("force-wildcard-resource") {
				forceWildcardResource, _ = cfg.Section("").Key("force-wildcard-resource").Bool()
			}
			if cfg.Section("").HasKey("aws-redirect-host") {
				awsRedirectHost = cfg.Section("").Key("aws-redirect-host").String()
			}
			if cfg.Section("").HasKey("gcp-discovery-parallel") {
				gcpDiscoveryParallel, _ = cfg.Section("").Key("gcp-discovery-parallel").Int()
			}

		}
	}

	providerFlag = flag.String("provider", provider, "the cloud service provider to intercept calls for")
	setiniFlag = flag.Bool("set-ini", setIni, "when set, the .aws/config file will be updated to use the CSM monitoring or CA bundle and removed when exiting")
	profileFlag = flag.String("profile", profile, "use the specified profile when combined with --set-ini")
	failsonlyFlag = flag.Bool("fails-only", failsOnly, "when set, only failed AWS calls will be added to the policy, csm mode only")
	outputFileFlag = flag.String("output-file", outputFile, "specify a file that will be written to on SIGHUP or exit")
	refreshRateFlag = flag.Int("refresh-rate", refreshRate, "instead of flushing to console every API call, do it this number of seconds")
	sortAlphabeticalFlag = flag.Bool("sort-alphabetical", sortAlphabetical, "sort actions alphabetically")
	hostFlag = flag.String("host", host, "host to listen on for CSM")
	modeFlag = flag.String("mode", mode, "the listening mode (csm,proxy)")
	bindAddrFlag = flag.String("bind-addr", bindAddr, "the bind address for proxy mode")
	caBundleFlag = flag.String("ca-bundle", caBundle, "the CA certificate bundle (PEM) to use for proxy mode")
	caKeyFlag = flag.String("ca-key", caKey, "the CA certificate key to use for proxy mode")
	accountIDFlag = flag.String("account-id", accountID, "the AWS account ID to use in policy outputs within proxy mode")
	backgroundFlag = flag.Bool("background", background, "when set, the process will return the current PID and run in the background without output")
	overrideAwsMapFlag = flag.String("override-aws-map", overrideAwsMap, "overrides the embedded AWS mapping JSON file with the filepath provided")
	debugFlag = flag.Bool("debug", debug, "dumps associated HTTP requests when set in proxy mode")
	forceWildcardResourceFlag = flag.Bool("force-wildcard-resource", forceWildcardResource, "when set, the Resource will always be a wildcard")
	csmPortFlag = flag.Int("csm-port", csmPort, "port to listen on for CSM")
	awsRedirectHostFlag = flag.String("aws-redirect-host", awsRedirectHost, "redirect all AWS API calls to this endpoint")
	gcpDiscoveryParallelFlag = flag.Int("gcp-discovery-parallel", gcpDiscoveryParallel, "number of parallel jobs to fetch GCP Discovery documents (default 10)")
}

func Run() {
	parseConfig()

	debugln("Flags parsed and configuration loaded")

	flag.Parse()

	// Create provider adapter
	provider := NewCloudProvider(*providerFlag)
	debugf("Selected provider: %s", provider.Name())

	// Force proxy mode for providers that don't support CSM
	if !provider.SupportsCSM() {
		debugln("Provider does not support CSM; forcing proxy mode")
		*modeFlag = "proxy"
	}

	if *backgroundFlag {
		args := os.Args[1:]
		for i := 0; i < len(args); i++ {
			if args[i] == "-background" || args[i] == "--background" {
				args = append(args[:i], args[i+1:]...)
				break
			}
		}
		cmd := exec.Command(os.Args[0], args...)
		cmd.Start()
		debugf("Running in background with PID %d", cmd.Process.Pid)
		fmt.Println(cmd.Process.Pid)
		os.Exit(0)
	}

	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
		debugf("CPU profiling enabled: %s", *cpuProfileFlag)
	}

	// Provider-specific pre-run setup (e.g., AWS refresh and INI)
	debugln("Starting provider PreRunSetup()")
	provider.PreRunSetup()
	debugln("Completed provider PreRunSetup()")

	debugln("Loading IAM/API maps")
	provider.LoadMaps()
	debugln("Loaded IAM/API maps")

	if *modeFlag == "csm" && provider.SupportsCSM() {
		debugln("Starting CSM mode")
		provider.RunCSM()
	} else if *modeFlag == "proxy" {
		debugln("Starting proxy mode: reading service files")
		provider.ReadServiceFiles()
		debugf("Launching proxy on %s", *bindAddrFlag)
		createProxy(*bindAddrFlag, *awsRedirectHostFlag)
	} else {
		fmt.Println("ERROR: unknown mode")
	}
}

func RunWithArgs(provider string, setIni bool, profile string, failsOnly bool, outputFile string, refreshRate int, sortAlphabetical bool, host, mode, bindAddr, caBundle, caKey, accountID string, background, debug, forceWildcardResource bool, awsRedirectHost string) {
	providerFlag = &provider
	setiniFlag = &setIni
	profileFlag = &profile
	failsonlyFlag = &failsOnly
	outputFileFlag = &outputFile
	refreshRateFlag = &refreshRate
	sortAlphabeticalFlag = &sortAlphabetical
	hostFlag = &host
	modeFlag = &mode
	bindAddrFlag = &bindAddr
	caBundleFlag = &caBundle
	caKeyFlag = &caKey
	accountIDFlag = &accountID
	backgroundFlag = &background
	debugFlag = &debug
	forceWildcardResourceFlag = &forceWildcardResource
	awsRedirectHostFlag = &awsRedirectHost

	// Create provider adapter
	prov := NewCloudProvider(*providerFlag)

	if *cpuProfileFlag != "" {
		f, err := os.Create(*cpuProfileFlag)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Provider-specific pre-run setup (e.g., AWS refresh and INI)
	prov.PreRunSetup()

	prov.LoadMaps()

	// Enforce supported modes
	if *modeFlag == "csm" && prov.SupportsCSM() {
		prov.RunCSM()
	} else if *modeFlag == "proxy" || !prov.SupportsCSM() {
		// fallback to proxy if CSM not supported
		prov.ReadServiceFiles()
		createProxy(*bindAddrFlag, *awsRedirectHostFlag)
	} else {
		fmt.Println("ERROR: unknown mode")
	}
}
