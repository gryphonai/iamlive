package iamlivecore

// ServiceDefinitionMetadata carries AWS model metadata.
type ServiceDefinitionMetadata struct {
	APIVersion          string `json:"apiVersion"`
	EndpointPrefix      string `json:"endpointPrefix"`
	JSONVersion         string `json:"jsonVersion"`
	Protocol            string `json:"protocol"`
	ServiceFullName     string `json:"serviceFullName"`
	ServiceAbbreviation string `json:"serviceAbbreviation"`
	ServiceID           string `json:"serviceId"`
	SignatureVersion    string `json:"signatureVersion"`
	TargetPrefix        string `json:"targetPrefix"`
	UID                 string `json:"uid"`
}
