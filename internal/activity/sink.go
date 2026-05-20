package activity

import "context"

// Sink publishes human-readable simple-mode progress (Dahusim SIMPLE_MODE_ACTIVITY).
type Sink func(ctx context.Context, text string)
