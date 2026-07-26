package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeEmailAlias_GmailAndGooglemailAliasesCollapse(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: " First.Last+tag@googlemail.com ", want: "firstlast@gmail.com"},
		{input: "firstlast@gmail.com", want: "firstlast@gmail.com"},
		{input: "first.last+tag@gmail.com", want: "firstlast@gmail.com"},
		{input: "Alice+promo@example.com", want: "alice@example.com"},
		{input: "not-an-email", want: "not-an-email"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, NormalizeEmailAlias(tt.input))
		})
	}
}

func TestUserSyncEmailAlias_TrimsEmailAndStoresNormalizedAlias(t *testing.T) {
	user := &User{Email: " First.Last+tag@googlemail.com "}

	user.SyncEmailAlias()

	require.Equal(t, "First.Last+tag@googlemail.com", user.Email)
	require.Equal(t, "firstlast@gmail.com", user.EmailAlias)
}
