package iamlivecore

// Entry is a single CSM entry.
type Entry struct {
	Region              string `json:"Region"`
	Type                string `json:"Type"`
	Service             string `json:"Service"`
	Method              string `json:"Api"`
	Parameters          map[string][]string
	URIParameters       map[string]string
	FinalHTTPStatusCode int    `json:"FinalHttpStatusCode"`
	AccessKey           string `json:"AccessKey"`
	SessionToken        string `json:"SessionToken"`
	Host                string `json:"_Host"`
}
