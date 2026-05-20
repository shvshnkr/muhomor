package apiclient

// ServiceState mirrors controller.ServiceState JSON.
type ServiceState string

const (
	StateIdle       ServiceState = "Idle"
	StateConnecting ServiceState = "Connecting"
	StateConnected  ServiceState = "Connected"
	StateStopping   ServiceState = "Stopping"
	StateStopped    ServiceState = "Stopped"
)

// ServiceStatus is GET /v1/service/status response.
type ServiceStatus struct {
	State              ServiceState `json:"State"`
	Connected          bool         `json:"Connected"`
	ProfileID          int64        `json:"ProfileID"`
	ProfileName        string       `json:"ProfileName"`
	ProxyName          string       `json:"ProxyName"`
	SubscriptionSource string       `json:"SubscriptionSource"`
}

func (s ServiceStatus) IsConnected() bool {
	return s.State == StateConnected
}

// JSONResponse is a generic daemon JSON body.
type JSONResponse map[string]any
