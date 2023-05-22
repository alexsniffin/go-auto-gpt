package messages

import (
	"github.com/google/uuid"
	"go-autogpt/pkg/models"
)

type NewGoal struct {
	RequestID uuid.UUID
	Goal      string
}

type NewPlan struct {
	RequestID uuid.UUID
	models.Plan
}

type NewSearch struct {
	RequestID           uuid.UUID
	Search              string
	ExpectedOutcome     string
	PossibleLimitations string
}

type NewTerminal struct {
	RequestID uuid.UUID
	Command   string
	Reason    string
	Task      string
}

type TerminalError struct {
	Command string `json:"command"`
	Error   string `json:"error"`
	Reason  string `json:"reason"`
}

type HandleTerminalError struct {
	Task             string          `json:"task"`
	PreviousAttempts []TerminalError `json:"previousAttempts"`
}

type SearchResult struct {
	Result string
}

type TerminalResult struct {
	Result string
}

type TaskResult struct {
	TaskHistory models.TaskHistory
}

type SupervisorComplete struct{}

type GetStatus struct{}

type ReportError struct {
	Error models.Error
}
