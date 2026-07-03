package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func newUsageIPGeoAdminService(t *testing.T, values map[string]string) *adminServiceImpl {
	t.Helper()
	settingSvc := NewSettingService(&modelCatalogSettingRepoStub{values: values}, &config.Config{})
	return &adminServiceImpl{
		settingService:  settingSvc,
		usageIPGeoCache: newUsageIPGeoCache(time.Hour),
	}
}

func TestAdminServiceLookupUsageIPGeoLocalStatuses(t *testing.T) {
	svc := newUsageIPGeoAdminService(t, map[string]string{
		SettingKeyUsageIPGeoEnabled: "false",
	})

	items, err := svc.LookupUsageIPGeo(context.Background(), []string{"8.8.8.8", "127.0.0.1", "not-an-ip"})
	require.NoError(t, err)
	require.Len(t, items, 3)
	require.Equal(t, "disabled", items[0].Status)
	require.Equal(t, "private", items[1].Status)
	require.Equal(t, "error", items[2].Status)
}

func TestAdminServiceLookupUsageIPGeoRejectsTooManyIPs(t *testing.T) {
	svc := newUsageIPGeoAdminService(t, map[string]string{})
	ips := make([]string, 51)
	for i := range ips {
		ips[i] = "8.8.8." + strconv.Itoa(i%255)
	}

	_, err := svc.LookupUsageIPGeo(context.Background(), ips)
	require.Error(t, err)
	var badRequest *infraerrors.ApplicationError
	require.ErrorAs(t, err, &badRequest)
	require.Equal(t, "USAGE_IP_GEO_TOO_MANY_IPS", badRequest.Reason)
}

func TestAdminServiceLookupUsageIPGeoProviderAndCache(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/ok/8.8.8.8":
			_, _ = w.Write([]byte(`{"country":"United States","region":"California","city":"Mountain View","asn":"AS15169"}`))
		case "/missing/8.8.4.4":
			w.WriteHeader(http.StatusNotFound)
		case "/empty/1.1.1.1":
			_, _ = w.Write([]byte(`{}`))
		case "/error/9.9.9.9":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer server.Close()

	svc := newUsageIPGeoAdminService(t, map[string]string{
		SettingKeyUsageIPGeoEnabled:     "true",
		SettingKeyUsageIPGeoProviderURL: server.URL + "/ok/{ip}",
		SettingKeyUsageIPGeoTimeoutMs:   "1000",
	})

	items, err := svc.LookupUsageIPGeo(context.Background(), []string{"8.8.8.8"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "ok", items[0].Status)
	require.Equal(t, "United States", items[0].Country)
	require.False(t, items[0].Cached)

	items, err = svc.LookupUsageIPGeo(context.Background(), []string{"8.8.8.8"})
	require.NoError(t, err)
	require.True(t, items[0].Cached)
	require.Equal(t, int64(1), calls.Load())

	for _, tt := range []struct {
		name   string
		path   string
		ip     string
		status string
	}{
		{name: "provider not found", path: "/missing/{ip}", ip: "8.8.4.4", status: "not_found"},
		{name: "empty payload", path: "/empty/{ip}", ip: "1.1.1.1", status: "not_found"},
		{name: "provider error", path: "/error/{ip}", ip: "9.9.9.9", status: "error"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := newUsageIPGeoAdminService(t, map[string]string{
				SettingKeyUsageIPGeoEnabled:     "true",
				SettingKeyUsageIPGeoProviderURL: server.URL + tt.path,
				SettingKeyUsageIPGeoTimeoutMs:   "1000",
			})
			items, err := svc.LookupUsageIPGeo(context.Background(), []string{tt.ip})
			require.NoError(t, err)
			require.Len(t, items, 1)
			require.Equal(t, tt.status, items[0].Status)
		})
	}
}
