package securityaudit

import "context"

type fakePromptRepo struct {
	nextID        int64
	jobs          []*Job
	events        []*Event
	deletePreview *DeletePreview
	deleted       *DeleteResult
	createJobErr  error
}

func (r *fakePromptRepo) CreateJob(_ context.Context, job *Job) error {
	if r.createJobErr != nil {
		return r.createJobErr
	}
	r.nextID++
	job.ID = r.nextID
	r.jobs = append(r.jobs, job)
	return nil
}

func (r *fakePromptRepo) CreateEvent(_ context.Context, job *Job, result *NormalizedResult, fullPrompt string) (*Event, error) {
	event := eventFromJob(job, result, fullPrompt)
	r.nextID++
	event.ID = r.nextID
	r.events = append(r.events, event)
	return event, nil
}

func (r *fakePromptRepo) ClaimJobs(context.Context, int) ([]*Job, error) { return nil, nil }
func (r *fakePromptRepo) MarkJobDone(context.Context, int64) error       { return nil }
func (r *fakePromptRepo) MarkJobFailed(context.Context, *Job, string, string, bool) error {
	return nil
}

func (r *fakePromptRepo) QueueStats(context.Context) (QueueStats, error) { return QueueStats{}, nil }

func (r *fakePromptRepo) ListEvents(context.Context, EventFilter) (*EventList, error) {
	items := make([]*Event, 0, len(r.events))
	for _, event := range r.events {
		clone := *event
		clone.FullPrompt = ""
		items = append(items, &clone)
	}
	return &EventList{Items: items, Total: int64(len(items)), Page: 1, PageSize: 20}, nil
}

func (r *fakePromptRepo) GetEvent(_ context.Context, id int64, includeFullText bool) (*Event, error) {
	for _, event := range r.events {
		if event.ID == id {
			clone := *event
			if !includeFullText {
				clone.FullPrompt = ""
			}
			return &clone, nil
		}
	}
	return nil, ErrEventNotFound
}

func (r *fakePromptRepo) DeleteEvent(context.Context, int64) (*DeleteResult, error) {
	return &DeleteResult{Deleted: 1, JobIDs: []int64{1}}, nil
}

func (r *fakePromptRepo) PreviewDelete(context.Context, EventFilter) (*DeletePreview, error) {
	if r.deletePreview != nil {
		return r.deletePreview, nil
	}
	return &DeletePreview{Matched: 1, SnapshotMaxID: 10}, nil
}

func (r *fakePromptRepo) DeleteEventsByFilter(context.Context, EventFilter, int64) (*DeleteResult, error) {
	if r.deleted != nil {
		return r.deleted, nil
	}
	return &DeleteResult{Deleted: 1, JobIDs: []int64{1}}, nil
}
