package models

import (
	"time"
)

type Planner struct {
	State       State               `json:"state"`
	TaskHistory []TaskHistory       `json:"history"`
	Plan        map[string][]string `json:"plan"`
	Errs        Error               `json:"error,omitempty"`
}

type Status struct {
	Planner Planner `json:"planner"`
}

type Error struct {
	ErrMessage string      `json:"error,omitempty"`
	Message    interface{} `json:"message,omitempty"`
	Time       *time.Time  `json:"time,omitempty"`
}

type Solution struct {
	Tool        string   `json:"tool"`
	Inputs      []string `json:"inputs"`
	Reasoning   string   `json:"reasoning"`
	Limitations string   `json:"limitations"`
	Outcome     string   `json:"outcome"`
}

type TaskHistory struct {
	Task     string   `json:"task"`
	Solution Solution `json:"solution"`
	Result   any      `json:"result"`
}

type Plan struct {
	Goal  string
	Tasks []string `json:"tasks"`
}
