package protocol

// Usage is the compact snapshot the widget renders.
type Usage struct {
	WeeklyPct        *float64 `json:"weekly_pct,omitempty"`
	WeeklyResetsAt   *int64   `json:"weekly_resets_at,omitempty"`
	FiveHourPct      *float64 `json:"five_hour_pct,omitempty"`
	FiveHourResetsAt *int64   `json:"five_hour_resets_at,omitempty"`
	ContextPct       *float64 `json:"context_pct,omitempty"`
	CostUSD          *float64 `json:"cost_usd,omitempty"`
	DurationMs       *int64   `json:"duration_ms,omitempty"`
	Model            string   `json:"model,omitempty"`
	Folder           string   `json:"folder,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	ReceivedAt       int64    `json:"received_at"`
}

type Notification struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	At        int64  `json:"at"`
}

// State is the single payload the widget polls from /v1/state.
type State struct {
	Usage         *Usage         `json:"usage"`
	Sessions      []Usage        `json:"sessions,omitempty"`
	Notifications []Notification `json:"notifications"`
	BridgeTime    int64          `json:"bridge_time"`
}
