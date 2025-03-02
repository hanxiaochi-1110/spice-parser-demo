package models

type Component struct {
	LineNum int                `json:"-"`
	Type    string             `json:"type"`
	Nodes   []string           `json:"nodes"`
	Model   string             `json:"model,omitempty"`
	Params  map[string]float64 `json:"params"`
}
