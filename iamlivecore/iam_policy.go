package iamlivecore

// IAMPolicy is a full IAM policy document.
type IAMPolicy struct {
	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}
