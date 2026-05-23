package model

import "github.com/muhomor/muhomor/internal/apiclient"

// BulkMemberUI is one PROXY_BULK leg for status tables.
type BulkMemberUI struct {
	Tag     string
	Name    string
	DelayMs int
	Error   string
}

// ConnectionUI state for Simple home screen.
type ConnectionUI struct {
	State        apiclient.ServiceState
	Connected    bool
	ProfileID    int64
	ProfileName  string
	ProxyName    string
	ActivityText string
	ProbeText      string
	MultipathText  string
	BulkMembers    []BulkMemberUI
	LastPingMs     int
	LastPingError  string
	Busy         bool
	ErrorText    string
}

// SettingsUI snapshot for tray/settings display.
type SettingsUI struct {
	ServiceMode              string
	MixedPort                int
	RouteQuick               int
	MultipathEnabled         bool
	MultipathPreset          string
	MultipathWLEmergencyOnly bool
	WLBuiltinConnectEnabled  bool
}
