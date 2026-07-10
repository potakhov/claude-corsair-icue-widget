package claudecode

// StatuslineInput is the JSON Claude Code pipes to a statusline command.
type StatuslineInput struct {
	SessionID string `json:"session_id"`
	Model     struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Cost struct {
		TotalCostUSD    float64 `json:"total_cost_usd"`
		TotalDurationMs int64   `json:"total_duration_ms"`
	} `json:"cost"`
	ContextWindow struct {
		UsedPercentage float64 `json:"used_percentage"`
	} `json:"context_window"`
	Workspace struct {
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	RateLimits *RateLimits `json:"rate_limits"`
}

type RateLimits struct {
	FiveHour *Window `json:"five_hour"`
	SevenDay *Window `json:"seven_day"`
}

type Window struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

// NotificationInput is the JSON a Notification/Stop hook receives on stdin.
type NotificationInput struct {
	SessionID        string `json:"session_id"`
	Message          string `json:"message"`
	Title            string `json:"title"`
	NotificationType string `json:"notification_type"`
	HookEventName    string `json:"hook_event_name"`
}
