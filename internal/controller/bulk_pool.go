package controller

import "github.com/muhomor/muhomor/internal/store"

// filterProfilesForBulk keeps only URL-alive legs from the latest pretest when that set is known.
func filterProfilesForBulk(primary store.Profile, legs []store.Profile, urlAlive []int64) (store.Profile, []store.Profile) {
	if len(urlAlive) == 0 {
		return primary, legs
	}
	alive := map[int64]struct{}{}
	for _, id := range urlAlive {
		alive[id] = struct{}{}
	}
	var filtered []store.Profile
	for _, p := range legs {
		if _, ok := alive[p.ID]; ok {
			filtered = append(filtered, p)
		}
	}
	if _, ok := alive[primary.ID]; ok {
		return primary, filtered
	}
	// Primary failed URL test but was ranked by TCP — bulk without it avoids sticky to a dead leg.
	if len(filtered) > 0 {
		return filtered[0], filtered[1:]
	}
	return primary, legs
}
