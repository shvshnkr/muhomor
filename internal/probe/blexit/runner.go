package blexit

import (
	"context"

	"github.com/muhomor/muhomor/internal/store"
)

// SuiteRunner runs BL-over-proxy URL suite (picker primary, ephemeral fallback).
type SuiteRunner interface {
	RunSuite(ctx context.Context, profiles []store.Profile, st *store.Store, urls []string, minPass int) (map[int64]bool, error)
	Name() string // "picker" | "ephemeral"
}
