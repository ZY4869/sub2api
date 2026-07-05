package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type redeemInvitationRepoStub struct {
	RedeemCodeRepository
	code *RedeemCode
}

func (r *redeemInvitationRepoStub) GetByCode(_ context.Context, _ string) (*RedeemCode, error) {
	return r.code, nil
}

func TestRedeemRejectsInvitationCodeAtUserRedeemEntry(t *testing.T) {
	svc := &RedeemService{
		redeemRepo: &redeemInvitationRepoStub{
			code: &RedeemCode{
				ID:     1,
				Code:   "INVITE-ONLY",
				Type:   RedeemTypeInvitation,
				Status: StatusUnused,
			},
		},
	}

	got, err := svc.Redeem(context.Background(), 42, "INVITE-ONLY")

	require.Nil(t, got)
	require.ErrorIs(t, err, ErrRedeemCodeUnsupportedType)
	require.Equal(t, "REDEEM_CODE_UNSUPPORTED_TYPE", infraerrors.Reason(err))
}
