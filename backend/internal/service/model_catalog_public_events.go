package service

import (
	"context"
	"sync"
)

const PublicModelCatalogPublishedEventName = "model_catalog.published"

type PublicModelCatalogPublishedEvent struct {
	ETag         string `json:"etag"`
	PublishedAt  string `json:"published_at,omitempty"`
	ModelCount   int    `json:"model_count"`
	ChangedCount int    `json:"changed_count"`
}

type PublicModelCatalogEvent struct {
	Name    string
	Payload PublicModelCatalogPublishedEvent
}

type PublicModelCatalogEventBroker struct {
	mu          sync.RWMutex
	subscribers map[chan PublicModelCatalogEvent]struct{}
}

func NewPublicModelCatalogEventBroker() *PublicModelCatalogEventBroker {
	return &PublicModelCatalogEventBroker{
		subscribers: make(map[chan PublicModelCatalogEvent]struct{}),
	}
}

func (b *PublicModelCatalogEventBroker) Subscribe(ctx context.Context) <-chan PublicModelCatalogEvent {
	ch := make(chan PublicModelCatalogEvent, 1)
	if b == nil {
		close(ch)
		return ch
	}

	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
		close(ch)
	}()

	return ch
}

func (b *PublicModelCatalogEventBroker) Publish(event PublicModelCatalogEvent) int {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()

	delivered := 0
	for ch := range b.subscribers {
		select {
		case ch <- event:
			delivered++
		default:
		}
	}
	return delivered
}

func (s *ModelCatalogService) SubscribePublicModelCatalogEvents(ctx context.Context) <-chan PublicModelCatalogEvent {
	if s == nil {
		ch := make(chan PublicModelCatalogEvent)
		close(ch)
		return ch
	}
	return s.publicCatalogEventBroker().Subscribe(ctx)
}

func (s *ModelCatalogService) PublishPublicModelCatalogEvent(payload PublicModelCatalogPublishedEvent) int {
	if s == nil {
		return 0
	}
	return s.publicCatalogEventBroker().Publish(PublicModelCatalogEvent{
		Name:    PublicModelCatalogPublishedEventName,
		Payload: payload,
	})
}

func (s *ModelCatalogService) publicCatalogEventBroker() *PublicModelCatalogEventBroker {
	s.publicCatalogEventMu.Lock()
	defer s.publicCatalogEventMu.Unlock()
	if s.publicCatalogEvents == nil {
		s.publicCatalogEvents = NewPublicModelCatalogEventBroker()
	}
	return s.publicCatalogEvents
}

func publicModelCatalogPublishedEventFromSummary(summary *PublicModelCatalogPublishedSummary) PublicModelCatalogPublishedEvent {
	if summary == nil {
		return PublicModelCatalogPublishedEvent{}
	}
	return PublicModelCatalogPublishedEvent{
		ETag:         summary.ETag,
		PublishedAt:  summary.PublishedAt,
		ModelCount:   summary.ModelCount,
		ChangedCount: summary.ChangedCount,
	}
}
