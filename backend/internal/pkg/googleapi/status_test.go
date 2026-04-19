//go:build unit

package googleapi

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPStatusToGoogleStatus_ServiceUnavailable(t *testing.T) {
	require.Equal(t, "UNAVAILABLE", HTTPStatusToGoogleStatus(http.StatusServiceUnavailable))
}
