package core

import "strings"

func ClassifyHarnessOutput(harness Harness, output string) string {
	lower := strings.ToLower(output)
	for _, pattern := range harness.ApprovalPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return "waiting_for_approval"
		}
	}
	for _, pattern := range harness.BusyPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return "running"
		}
	}
	if strings.TrimSpace(output) == "" {
		return "unknown"
	}
	return "idle"
}
