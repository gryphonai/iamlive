package iamlivecore

// AzureIAMPolicy represents a minimal Azure role definition output by iamlive.
type AzureIAMPolicy struct {
	Name             string   `json:"Name"`
	IsCustom         bool     `json:"IsCustom"`
	Description      string   `json:"Description"`
	Actions          []string `json:"Actions"`
	DataActions      []string `json:"DataActions"`
	NotDataActions   []string `json:"NotDataActions"`
	AssignableScopes []string `json:"AssignableScopes"`
}
