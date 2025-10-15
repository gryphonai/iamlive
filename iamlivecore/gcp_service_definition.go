package iamlivecore

// GCPServiceDefinition represents a discovery document for a GCP API service.
type GCPServiceDefinition struct {
	RootURL    string `json:"rootUrl"`
	BasePath   string `json:"basePath"`
	RootDomain string
	Resources  map[string]GCPResourceDefinition `json:"resources"`
}
