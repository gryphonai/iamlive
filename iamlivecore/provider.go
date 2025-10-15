package iamlivecore

import "net/http"

// CloudProvider defines the behavior for a supported cloud.
// We encapsulate cloud-specific logic behind this interface to reduce
// conditionals in service.go and proxy.go and move toward SOLID.
type CloudProvider interface {
	// Name returns the canonical provider name (e.g., "aws", "azure", "gcp").
	Name() string
	// SupportsCSM indicates whether CSM mode is supported for this provider.
	SupportsCSM() bool
	// LoadMaps loads any embedded policy/API maps required by the provider.
	LoadMaps()
	// PreRunSetup performs provider-specific runtime setup (e.g., AWS INI / refresh).
	PreRunSetup()
	// ReadServiceFiles loads any provider-specific service definitions used by proxy mode.
	ReadServiceFiles()
	// HandleHTTPRequest processes an HTTP request in proxy mode if it belongs to this provider.
	// Returns processed=true and possibly a modified body if handled; otherwise processed=false.
	HandleHTTPRequest(req *http.Request, awsRedirectHost string) (processed bool, body []byte)
	// RunCSM starts provider-specific CSM mode if supported (AWS only currently).
	RunCSM()
	// HostnamePattern returns a regex snippet (no anchors) matching this provider's API hostnames.
	HostnamePattern() string
}

func NewCloudProvider(name string) CloudProvider {
	switch name {
	case "aws":
		return awsProvider{}
	case "azure":
		return azureProvider{}
	case "gcp":
		return gcpProvider{}
	default:
		// Fallback to AWS behavior to preserve existing defaults
		return awsProvider{}
	}
}
