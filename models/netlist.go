package models

type Netlist struct {
	Title      string            `json:"title"`
	Components []*Component      `json:"components"`
	Commands   []*Command        `json:"commands"`
	Models     map[string]*Model `json:"models"`
}
