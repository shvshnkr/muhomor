package apiclient

import "github.com/muhomor/muhomor/internal/api"

// Re-export API DTOs for clients.
type (
	ServiceState       = api.ServiceState
	ServiceStatus      = api.ServiceStatus
	Settings           = api.Settings
	Profile            = api.Profile
	ImportRequest      = api.ImportRequest
	ImportResult       = api.ImportResult
	ProfileEnabledReq  = api.ProfileEnabledRequest
	PingResponse       = api.PingResponse
	Event              = api.Event
	ProbeProgress      = api.ProbeProgress
	MultipathProgress  = api.MultipathProgress
	JSONResponse       = map[string]any
)

const (
	StateIdle       = api.StateIdle
	StateConnecting = api.StateConnecting
	StateConnected  = api.StateConnected
	StateStopping   = api.StateStopping
	StateStopped    = api.StateStopped
)
