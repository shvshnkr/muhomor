package aggregate

// FlowClass traffic class for scheduling policy.
type FlowClass int

const (
	FlowInteractive FlowClass = iota
	FlowBulk
	FlowBackground
	FlowSingleStream
)

func (f FlowClass) String() string {
	switch f {
	case FlowBulk:
		return "bulk"
	case FlowBackground:
		return "background"
	case FlowSingleStream:
		return "single_stream"
	default:
		return "interactive"
	}
}

// ClassifyFlow picks policy bucket (desktop heuristic without deep DPI).
func ClassifyFlow(opts ClassifyOpts) FlowClass {
	if opts.SingleStreamHint {
		return FlowSingleStream
	}
	if opts.BackgroundHint {
		return FlowBackground
	}
	if opts.BulkHint || opts.ExpectedMbps >= 50 {
		return FlowBulk
	}
	return FlowInteractive
}

// ClassifyOpts hints for flow classification.
type ClassifyOpts struct {
	SingleStreamHint bool
	BulkHint         bool
	BackgroundHint   bool
	ExpectedMbps     int
}
