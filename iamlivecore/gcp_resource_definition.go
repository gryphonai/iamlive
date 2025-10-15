package iamlivecore

type GCPResourceDefinition struct {
	Methods   map[string]GCPMethodDefinition   `json:"methods"`
	Resources map[string]GCPResourceDefinition `json:"resources"`
}
