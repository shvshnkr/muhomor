package controller

// ServiceState mirrors Dahusim ServiceState (subset).
type ServiceState string

const (
	StateIdle       ServiceState = "Idle"
	StateConnecting ServiceState = "Connecting"
	StateConnected  ServiceState = "Connected"
	StateStopping   ServiceState = "Stopping"
	StateStopped    ServiceState = "Stopped"
)

type Status struct {
	State              ServiceState
	Connected          bool
	ProfileID          int64
	ProfileName        string
	ProxyName          string
	SubscriptionSource string
}

func (s Status) ConnectedBool() bool {
	return s.State == StateConnected
}
