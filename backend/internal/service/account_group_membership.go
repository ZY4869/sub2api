package service

func accountBoundToGroupID(account *Account, groupID *int64) bool {
	if account == nil {
		return false
	}
	if groupID == nil {
		return len(account.AccountGroups) == 0 && len(account.GroupIDs) == 0
	}
	if *groupID <= 0 {
		return false
	}
	for _, ag := range account.AccountGroups {
		if ag.GroupID == *groupID {
			return true
		}
	}
	for _, id := range account.GroupIDs {
		if id == *groupID {
			return true
		}
	}
	return false
}
