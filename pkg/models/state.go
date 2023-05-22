package models

type State string

const (
	Init     State = "init"
	Thinking State = "thinking"
	Idle     State = "idle"
	Failed   State = "failed" // dead state
	Finished State = "finished"
)
