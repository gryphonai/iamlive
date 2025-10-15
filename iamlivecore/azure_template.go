package iamlivecore

// AzureTemplate represents an ARM template.
type AzureTemplate struct {
	Resources []AzureTemplateResource `json:"resources"`
}
