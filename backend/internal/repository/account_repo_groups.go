package repository

import (
	"context"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbgroup "github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *accountRepository) AddToGroup(ctx context.Context, accountID, groupID int64, priority int) error {
	_, err := r.client.AccountGroup.Create().SetAccountID(accountID).SetGroupID(groupID).SetPriority(priority).Save(ctx)
	if err != nil {
		return err
	}
	payload := buildSchedulerGroupPayload([]int64{groupID})
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountGroupsChanged, &accountID, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue add to group failed: account=%d group=%d err=%v", accountID, groupID, err)
	}
	return nil
}
func (r *accountRepository) RemoveFromGroup(ctx context.Context, accountID, groupID int64) error {
	_, err := r.client.AccountGroup.Delete().Where(dbaccountgroup.AccountIDEQ(accountID), dbaccountgroup.GroupIDEQ(groupID)).Exec(ctx)
	if err != nil {
		return err
	}
	payload := buildSchedulerGroupPayload([]int64{groupID})
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountGroupsChanged, &accountID, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue remove from group failed: account=%d group=%d err=%v", accountID, groupID, err)
	}
	return nil
}
func (r *accountRepository) GetGroups(ctx context.Context, accountID int64) ([]service.Group, error) {
	groups, err := r.client.Group.Query().Where(dbgroup.HasAccountsWith(dbaccount.IDEQ(accountID))).All(ctx)
	if err != nil {
		return nil, err
	}
	outGroups := make([]service.Group, 0, len(groups))
	for i := range groups {
		outGroups = append(outGroups, *groupEntityToService(groups[i]))
	}
	return outGroups, nil
}
