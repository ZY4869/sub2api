package grokoauth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAuthorizationInput(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantKind  AuthorizationInputKind
		wantCode  string
		wantState string
		wantUser  string
	}{
		{
			name:      "callback url",
			raw:       "http://127.0.0.1:56121/callback?code=auth-code&state=state-1",
			wantKind:  AuthorizationInputCallbackURL,
			wantCode:  "auth-code",
			wantState: "state-1",
		},
		{
			name:      "query string",
			raw:       "?code=auth-code&state=state-1",
			wantKind:  AuthorizationInputQueryString,
			wantCode:  "auth-code",
			wantState: "state-1",
		},
		{
			name:     "bare code",
			raw:      "oauth-auth-code-with-lowercase",
			wantKind: AuthorizationInputBareAuthCode,
			wantCode: "oauth-auth-code-with-lowercase",
		},
		{
			name:     "device url",
			raw:      "https://auth.x.ai/activate?user_code=abcd-efgh",
			wantKind: AuthorizationInputDeviceURL,
			wantUser: "ABCD-EFGH",
		},
		{
			name:     "device short code",
			raw:      "abcd-efgh",
			wantKind: AuthorizationInputDeviceUserCode,
			wantUser: "ABCD-EFGH",
		},
		{
			name:     "empty",
			raw:      " ",
			wantKind: AuthorizationInputUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAuthorizationInput(tt.raw)
			require.Equal(t, tt.wantKind, got.Kind)
			require.Equal(t, tt.wantCode, got.Code)
			require.Equal(t, tt.wantState, got.State)
			require.Equal(t, tt.wantUser, got.UserCode)
		})
	}
}
