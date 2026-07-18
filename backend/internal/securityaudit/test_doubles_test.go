package securityaudit

import (
	"context"
	"errors"
	"time"
)

type fakeScanner struct {
	result *NormalizedResult
	err    error
	calls  int
}

func (s *fakeScanner) Scan(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type memoryPayloadStore struct {
	err     error
	values  map[int64]PromptPayload
	deleted []int64
}

func newMemoryPayloadStore() *memoryPayloadStore {
	return &memoryPayloadStore{values: map[int64]PromptPayload{}}
}

func (s *memoryPayloadStore) Set(_ context.Context, jobID int64, payload PromptPayload, _ time.Duration) error {
	if s.err != nil {
		return s.err
	}
	s.values[jobID] = payload
	return nil
}

func (s *memoryPayloadStore) Get(_ context.Context, jobID int64) (PromptPayload, error) {
	payload, ok := s.values[jobID]
	if !ok {
		return PromptPayload{}, errors.New("missing payload")
	}
	return payload, nil
}

func (s *memoryPayloadStore) Delete(_ context.Context, jobID int64) error {
	delete(s.values, jobID)
	s.deleted = append(s.deleted, jobID)
	return nil
}

func (s *memoryPayloadStore) Ping(context.Context) error { return nil }

type memoryConfigInvalidator struct {
	publishCalls int
	handler      func()
}

func (i *memoryConfigInvalidator) Publish(context.Context) error {
	i.publishCalls++
	if i.handler != nil {
		i.handler()
	}
	return nil
}

func (i *memoryConfigInvalidator) Subscribe(_ context.Context, handler func()) error {
	i.handler = handler
	return nil
}
