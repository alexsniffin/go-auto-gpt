package handler

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/chains"
	"go-autogpt/pkg/messages"
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

func (h *Handler) Plan(ctx context.Context, newAction messages.NewGoal) models.HandlerResult {
	completion, err := chains.Call(ctx, h.chain, map[string]any{"Goal": newAction.Goal})
	if err != nil {
		return models.HandlerResult{Error: fmt.Errorf("call: %w", err)}
	}

	question, err := template.Parse(prompts.PlanTemplate, newAction)
	if err != nil {
		return models.HandlerResult{Error: fmt.Errorf("execute: %w", err)}
	}

	return models.HandlerResult{Question: question, Answer: completion["text"].(string)}
}
