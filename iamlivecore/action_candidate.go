package iamlivecore

// ActionCandidate represents a potential AWS action resolved from a request.
type ActionCandidate struct {
	Path      string
	Action    string
	URIParams map[string]string
	Params    map[string][]string
	Operation ServiceOperation
	Service   string
}
