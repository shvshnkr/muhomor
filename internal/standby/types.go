package standby

import "time"

const (
	HotSize             = 3
	WarmSize            = 8
	CandidateBatchCap   = 12
	HotMaxAge           = 5 * time.Minute
	WarmMaxAge          = 20 * time.Minute
	HotRefreshInterval  = 60 * time.Second
	WarmRefreshInterval = 3 * time.Minute
)

// Entry is one URL-verified standby proxy.
type Entry struct {
	ProfileID  int64     `json:"profile_id"`
	ProxyName  string    `json:"proxy_name,omitempty"`
	DelayMs    int       `json:"delay_ms"`
	BLDelayMs  int       `json:"bl_delay_ms,omitempty"`
	VerifiedAt time.Time `json:"verified_at"`
}
