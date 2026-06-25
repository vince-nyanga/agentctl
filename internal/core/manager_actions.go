package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

const actionBlockStart = "AGENTCTL_ACTIONS:"
const actionBlockEnd = "END_AGENTCTL_ACTIONS"

type ManagerAction struct {
	Type              string `json:"type"`
	AgentName         string `json:"agent_name,omitempty"`
	Message           string `json:"message,omitempty"`
	ApprovalType      string `json:"approval_type,omitempty"`
	Title             string `json:"title,omitempty"`
	Description       string `json:"description,omitempty"`
	Risk              string `json:"risk,omitempty"`
	RecommendedAction string `json:"recommended_action,omitempty"`
}

func ParseManagerActions(text string) ([]ManagerAction, error) {
	start := strings.LastIndex(text, actionBlockStart)
	if start == -1 {
		return nil, nil
	}
	start += len(actionBlockStart)
	end := strings.Index(text[start:], actionBlockEnd)
	if end == -1 {
		return nil, fmt.Errorf("manager action block missing %s", actionBlockEnd)
	}
	payload := strings.TrimSpace(text[start : start+end])
	if payload == "" {
		return nil, nil
	}
	var actions []ManagerAction
	if err := json.Unmarshal([]byte(payload), &actions); err != nil {
		return nil, fmt.Errorf("parse manager actions: %w", err)
	}
	for _, action := range actions {
		switch action.Type {
		case "approval", "nudge", "done":
		default:
			return nil, fmt.Errorf("unsupported manager action type %q", action.Type)
		}
	}
	return actions, nil
}
