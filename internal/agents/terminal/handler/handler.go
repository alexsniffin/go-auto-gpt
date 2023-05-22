package handler

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/chains"
	"go-autogpt/pkg/models"
	"go-autogpt/pkg/prompts"
	"go-autogpt/pkg/template"
	"os"
	"os/exec"
)

type Handler struct {
	chain chains.Chain
}

func New(chain chains.Chain) *Handler {
	return &Handler{
		chain: chain,
	}
}

type input struct {
	Task             string
	PreviousAttempts string
}

func (h *Handler) RunCommand(ctx context.Context, command, id string) (string, error) {
	output, err := executeCommand(command, id)
	if err != nil {
		return "", err
	}

	return output, nil
}

func (h *Handler) DiagnoseNextAttempt(ctx context.Context, task, previousAttempts string) models.HandlerResult {
	completion, err := chains.Call(ctx, h.chain, map[string]any{"Task": task, "PreviousAttempts": previousAttempts})
	if err != nil {
		return models.HandlerResult{Error: fmt.Errorf("call: %w", err)}
	}

	question, err := template.Parse(prompts.TerminalDiagnoseError, input{
		Task:             task,
		PreviousAttempts: previousAttempts,
	})
	if err != nil {
		return models.HandlerResult{Error: fmt.Errorf("execute: %w", err)}
	}

	return models.HandlerResult{Question: question, Answer: completion["text"].(string)}
}

func executeCommand(command, id string) (string, error) {
	cmd := exec.Command("bash", "-c", "cd sandbox/"+id+" && "+command)

	output, err := cmd.CombinedOutput()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return "", fmt.Errorf("output=[%s], process state=[%s], error=[%w]", output, cmd.ProcessState.String(), err)
	}

	return string(output), nil
}

func (h *Handler) CreateDirectoryIfNotExists(id string) error {
	_, err := os.Stat(id)
	if os.IsNotExist(err) {
		err := os.MkdirAll("sandbox/"+id+"/tmp/", os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
