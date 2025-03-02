package models

type Command struct {
	Type    string            `json:"type"`
	Options map[string]string `json:"options"`
}
