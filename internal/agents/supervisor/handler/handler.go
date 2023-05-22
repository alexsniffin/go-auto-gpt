package handler

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/chains"
	"go-autogpt/pkg/models"
	"go-autogpt/pkg/prompts"
	"go-autogpt/pkg/template"
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
	Goal    string
	Task    string
	History string
}

func (h *Handler) Solution(ctx context.Context, task, goal, history string) models.HandlerResult {
	completion, err := chains.Call(ctx, h.chain, map[string]any{"Task": task, "Goal": goal, "History": history})
	if err != nil {
		return models.HandlerResult{Error: fmt.Errorf("call: %w", err)}
	}

	input := input{Goal: goal, Task: task, History: history}
	question, err := template.Parse(prompts.AgentTaskTemplate, input)
	if err != nil {
		return models.HandlerResult{Error: fmt.Errorf("execute: %w", err)}
	}

	return models.HandlerResult{
		Question: question,
		Answer:   completion["text"].(string),
	}
}
