package models

type Model struct {
	Name   string             `json:"name"`
	Type   string             `json:"type"`
	Params map[string]float64 `json:"params"`
}
