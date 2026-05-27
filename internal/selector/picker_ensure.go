package selector

import (
	"context"

	"github.com/muhomor/muhomor/internal/mihomo"
)

// pickerReady returns true when the long-lived picker subprocess is running (Ensure is idempotent).
func pickerReady(ctx context.Context, picker PickerTester) bool {
	if picker == nil {
		return false
	}
	if p, ok := picker.(*mihomo.Picker); ok {
		return p.Ensure(ctx) == nil
	}
	if err := picker.Ensure(ctx); err != nil {
		return false
	}
	return picker.Available()
}
