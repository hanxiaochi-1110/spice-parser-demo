package models

type ParseError struct {
	LineNum  int    `json:"line"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // WARNING/ERROR
}
