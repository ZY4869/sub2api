package servertiming

import "net/http"

type roundTripper struct {
	next http.RoundTripper
}

func (rt roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || !Active(req.Context()) {
		return rt.next.RoundTrip(req)
	}
	done := Observe(req.Context(), "http")
	resp, err := rt.next.RoundTrip(req)
	done()
	return resp, err
}
