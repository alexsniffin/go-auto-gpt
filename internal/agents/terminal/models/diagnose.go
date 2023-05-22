package models

type Diagnose struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}
