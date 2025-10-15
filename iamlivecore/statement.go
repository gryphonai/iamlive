package iamlivecore

// Statement is a single statement within an IAM policy.
type Statement struct {
	Effect   string      `json:"Effect"`
	Action   []string    `json:"Action"`
	Resource interface{} `json:"Resource"`
}
