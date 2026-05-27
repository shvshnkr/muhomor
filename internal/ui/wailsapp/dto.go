package wailsapp

// ConnectionDTO is JSON-safe connection state for the React UI.
type ConnectionDTO struct {
	State         string `json:"state"`
	Connected     bool   `json:"connected"`
	ConnectedVerified bool `json:"connectedVerified"`
	ConnectedDegraded bool `json:"connectedDegraded"`
	VerificationPhase string `json:"verificationPhase"`
	ProfileID     int64  `json:"profileId"`
	ProfileName   string `json:"profileName"`
	ProxyName     string `json:"proxyName"`
	StatusTitle   string `json:"statusTitle"`
	ActivityText  string `json:"activityText"`
	ProbeText     string `json:"probeText"`
	Busy          bool   `json:"busy"`
	Connecting    bool   `json:"connecting"`
	ErrorText     string `json:"errorText"`
	ConnectLabel  string `json:"connectLabel"`
	ConnectDanger bool   `json:"connectDanger"`
	PingLabel     string `json:"pingLabel"`
	PingResult    string `json:"pingResult"`
	PingError     bool   `json:"pingError"`
	PingEnabled   bool   `json:"pingEnabled"`
	TrafficUp     int64  `json:"trafficUp"`
	TrafficDown   int64  `json:"trafficDown"`
}

// SettingsDTO is a settings snapshot for extended tabs.
type SettingsDTO struct {
	ServiceMode              string `json:"serviceMode"`
	MixedPort                int    `json:"mixedPort"`
	RouteQuick               int    `json:"routeQuick"`
	MultipathEnabled         bool   `json:"multipathEnabled"`
	MultipathPreset          string `json:"multipathPreset"`
	MultipathWLEmergencyOnly bool   `json:"multipathWlEmergencyOnly"`
	WLBuiltinConnectEnabled  bool   `json:"wlBuiltinConnectEnabled"`
	UIKeepErrorsOnScreen     bool   `json:"uiKeepErrorsOnScreen"`
}

// PingResultDTO is returned from Ping bindings.
type PingResultDTO struct {
	Text  string `json:"text"`
	Error bool   `json:"error"`
}

// GroupDTO is a subscription group row.
type GroupDTO struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	SubscriptionLink string `json:"subscriptionLink"`
	Label            string `json:"label"`
}

// ProfileDTO is a profile row.
type ProfileDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	GroupID     int64  `json:"groupId"`
	Enabled     bool   `json:"enabled"`
	LastDelayMs int    `json:"lastDelayMs"`
}
