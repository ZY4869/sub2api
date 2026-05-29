package service

// buildSelectedSet converts a slice of selected CRS account IDs to a set for O(1) lookup.
// Returns nil if ids is nil (field not sent: backward compatible, create all).
// Returns an empty map if ids is non-nil but empty (user selected none: create none).
func buildSelectedSet(ids []string) map[string]struct{} {
	if ids == nil {
		return nil
	}
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// shouldCreateAccount checks if a new CRS account should be created based on user selection.
// Returns true if selectedSet is nil (backward compatible: create all) or if crsID is in the set.
func shouldCreateAccount(crsID string, selectedSet map[string]struct{}) bool {
	if selectedSet == nil {
		return true
	}
	_, ok := selectedSet[crsID]
	return ok
}
