//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthSnapshotRoundTripPreservesGroupPriority(t *testing.T) {
	svc := &APIKeyService{}
	apiKey := &APIKey{
		ID:     1,
		UserID: 7,
		Key:    "sk-test",
		Status: StatusActive,
		User: &User{
			ID:     7,
			Status: StatusActive,
			Role:   RoleAdmin,
		},
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 10,
				Group: &Group{
					ID:       10,
					Name:     "primary",
					Platform: PlatformAnthropic,
					Priority: 1,
					Status:   StatusActive,
				},
			},
			{
				GroupID: 20,
				Group: &Group{
					ID:       20,
					Name:     "fallback",
					Platform: PlatformAnthropic,
					Priority: 3,
					Status:   StatusActive,
				},
			},
		},
	}
	apiKey.SyncLegacyGroupShadow()

	snapshot := svc.snapshotFromAPIKey(apiKey)
	require.NotNil(t, snapshot)
	require.Len(t, snapshot.Groups, 2)
	require.Equal(t, 1, snapshot.Groups[0].Group.Priority)
	require.Equal(t, 3, snapshot.Groups[1].Group.Priority)

	restored := svc.snapshotToAPIKey(apiKey.Key, snapshot)
	require.NotNil(t, restored)
	require.Len(t, restored.GroupBindings, 2)
	require.NotNil(t, restored.GroupBindings[0].Group)
	require.NotNil(t, restored.GroupBindings[1].Group)
	require.Equal(t, 1, restored.GroupBindings[0].Group.Priority)
	require.Equal(t, 3, restored.GroupBindings[1].Group.Priority)
	require.NotNil(t, restored.GroupID)
	require.Equal(t, int64(10), *restored.GroupID)
}
