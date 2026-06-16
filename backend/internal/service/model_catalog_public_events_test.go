//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublicModelCatalogEventBroker_PublishesPublishedEvent(t *testing.T) {
	broker := NewPublicModelCatalogEventBroker()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := broker.Subscribe(ctx)
	delivered := broker.Publish(PublicModelCatalogEvent{
		Name: PublicModelCatalogPublishedEventName,
		Payload: PublicModelCatalogPublishedEvent{
			ETag:         `W/"etag-1"`,
			PublishedAt:  "2026-06-16T10:00:00Z",
			ModelCount:   2,
			ChangedCount: 1,
		},
	})

	require.Equal(t, 1, delivered)
	select {
	case event := <-events:
		require.Equal(t, PublicModelCatalogPublishedEventName, event.Name)
		require.Equal(t, `W/"etag-1"`, event.Payload.ETag)
		require.Equal(t, "2026-06-16T10:00:00Z", event.Payload.PublishedAt)
		require.Equal(t, 2, event.Payload.ModelCount)
		require.Equal(t, 1, event.Payload.ChangedCount)
	case <-time.After(time.Second):
		t.Fatal("expected published event")
	}
}

func TestPublicModelCatalogPublishedEventFromSummary_UsesWhitelistPayload(t *testing.T) {
	event := publicModelCatalogPublishedEventFromSummary(&PublicModelCatalogPublishedSummary{
		ETag:              `W/"etag-2"`,
		UpdatedAt:         "2026-06-16T09:59:00Z",
		PublishedAt:       "2026-06-16T10:00:00Z",
		LastRevalidatedAt: "2026-06-16T10:01:00Z",
		StaleReason:       "ignored",
		PageSize:          20,
		ModelCount:        3,
		ChangedCount:      2,
	})

	require.Equal(t, PublicModelCatalogPublishedEvent{
		ETag:         `W/"etag-2"`,
		PublishedAt:  "2026-06-16T10:00:00Z",
		ModelCount:   3,
		ChangedCount: 2,
	}, event)
}
