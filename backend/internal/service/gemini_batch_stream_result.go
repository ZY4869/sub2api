package service

import (
	"io"
	"net/http"
)

type GoogleBatchUpstreamResult interface {
	googleBatchUpstreamResult()
}

func (*UpstreamHTTPResult) googleBatchUpstreamResult() {}

type UpstreamHTTPStreamResult struct {
	StatusCode    int
	Headers       http.Header
	Body          io.ReadCloser
	ContentLength int64
}

func (*UpstreamHTTPStreamResult) googleBatchUpstreamResult() {}
