package model

import "github.com/muhomor/muhomor/internal/apiclient"

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
