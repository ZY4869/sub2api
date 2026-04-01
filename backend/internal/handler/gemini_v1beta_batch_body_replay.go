package handler

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type replayableGoogleBatchBody struct {
	mu          sync.Mutex
	source      io.ReadCloser
	spoolFile   *os.File
	spoolPath   string
	firstOpened bool
	completed   bool
	failed      error
	cleanupOnce sync.Once
}

func newReplayableGoogleBatchBody(source io.ReadCloser) (*replayableGoogleBatchBody, error) {
	spoolFile, err := os.CreateTemp("", "sub2api-google-batch-*")
	if err != nil {
		return nil, err
	}
	return &replayableGoogleBatchBody{
		source:    source,
		spoolFile: spoolFile,
		spoolPath: spoolFile.Name(),
	}, nil
}

func (b *replayableGoogleBatchBody) Open() (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failed != nil {
		return nil, b.failed
	}
	if !b.firstOpened {
		b.firstOpened = true
		return &replayableGoogleBatchBodyReader{parent: b, source: b.source, spoolFile: b.spoolFile}, nil
	}
	if !b.completed {
		return nil, fmt.Errorf("google batch request body replay not ready")
	}
	return os.Open(b.spoolPath)
}

func (b *replayableGoogleBatchBody) Cleanup() {
	if b == nil {
		return
	}
	b.cleanupOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.source != nil {
			_ = b.source.Close()
			b.source = nil
		}
		if b.spoolFile != nil {
			_ = b.spoolFile.Close()
			b.spoolFile = nil
		}
		if b.spoolPath != "" {
			_ = os.Remove(b.spoolPath)
		}
	})
}

func (b *replayableGoogleBatchBody) markCompleted() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.spoolFile == nil {
		return fmt.Errorf("google batch request spool unavailable")
	}
	if err := b.spoolFile.Sync(); err != nil {
		b.failed = err
		return err
	}
	if err := b.spoolFile.Close(); err != nil {
		b.failed = err
		return err
	}
	b.spoolFile = nil
	b.completed = true
	b.source = nil
	return nil
}

func (b *replayableGoogleBatchBody) markFailed(err error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err == nil {
		err = fmt.Errorf("google batch request body replay failed")
	}
	b.failed = err
	if b.spoolFile != nil {
		_ = b.spoolFile.Close()
		b.spoolFile = nil
	}
	return err
}

type replayableGoogleBatchBodyReader struct {
	parent    *replayableGoogleBatchBody
	source    io.ReadCloser
	spoolFile *os.File
	finished  bool
	closeOnce sync.Once
}

func (r *replayableGoogleBatchBodyReader) Read(p []byte) (int, error) {
	if r == nil || r.source == nil {
		return 0, io.EOF
	}
	n, err := r.source.Read(p)
	if n > 0 && r.spoolFile != nil {
		if _, writeErr := r.spoolFile.Write(p[:n]); writeErr != nil {
			return n, r.parent.markFailed(writeErr)
		}
	}
	if err == io.EOF {
		r.finished = true
		if completeErr := r.parent.markCompleted(); completeErr != nil {
			return n, completeErr
		}
	}
	if err != nil && err != io.EOF {
		return n, r.parent.markFailed(err)
	}
	return n, err
}

func (r *replayableGoogleBatchBodyReader) Close() error {
	if r == nil {
		return nil
	}
	var closeErr error
	r.closeOnce.Do(func() {
		if r.source != nil {
			closeErr = r.source.Close()
		}
		if !r.finished {
			closeErr = r.parent.markFailed(fmt.Errorf("google batch request body closed before replay completed"))
		}
	})
	return closeErr
}
