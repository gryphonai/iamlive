package iamlivecore

// AzureEntry captures an Azure REST call made via the proxy.
type AzureEntry struct {
	HTTPMethod string
	Path       string
	Parameters map[string][]string
	Body       []byte
}
