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

type ExecuteCommand struct {
	RequestID        uuid.UUID
	Command          string
	Reason           string
	Task             string
	PreviousAttempts []CommandAttempt `json:"previousAttempts"`
}

type CommandAttempt struct {
	Command string `json:"command"`
	Output  string `json:"output"`
	Error   string `json:"error"`
	Reason  string `json:"reason"`
}

type DiagnoseCommand struct {
	Task             string           `json:"task"`
	PreviousAttempts []CommandAttempt `json:"previousAttempts"`
}

type SearchResult struct {
	Result string
}

type CommandResult struct {
	Result             string           `json:"result"`
	DiagnosticAttempts []CommandAttempt `json:"diagnosticAttempts,omitempty"`
}

type TaskResult struct {
	TaskHistory models.TaskHistory
}

type SupervisorComplete struct {
	Result any
}

type GetStatus struct{}

type ReportError struct {
	Error models.Error
}
