package service

import (
	"context"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// SubscriptionExpiryService periodically updates expired subscription status.
type SubscriptionExpiryService struct {
	userSubRepo    UserSubscriptionRepository
	userRepo       UserRepository
	emailService   *EmailService
	emailTemplates *EmailTemplateService
	settingService *SettingService
	interval       time.Duration
	stopCh         chan struct{}
	stopOnce       sync.Once
	wg             sync.WaitGroup
}

func NewSubscriptionExpiryService(userSubRepo UserSubscriptionRepository, interval time.Duration) *SubscriptionExpiryService {
	return &SubscriptionExpiryService{
		userSubRepo: userSubRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

func (s *SubscriptionExpiryService) SetNotificationServices(emailService *EmailService, templates *EmailTemplateService, userRepo UserRepository, settings *SettingService) {
	s.emailService = emailService
	s.emailTemplates = templates
	s.userRepo = userRepo
	s.settingService = settings
}

func (s *SubscriptionExpiryService) Start() {
	if s == nil || s.userSubRepo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *SubscriptionExpiryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *SubscriptionExpiryService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	updated, err := s.userSubRepo.BatchUpdateExpiredStatus(ctx)
	if err != nil {
		log.Printf("[SubscriptionExpiry] Update expired subscriptions failed: %v", err)
		return
	}
	if updated > 0 {
		log.Printf("[SubscriptionExpiry] Updated %d expired subscriptions", updated)
	}
	s.sendExpiringNotifications(ctx)
}

func (s *SubscriptionExpiryService) sendExpiringNotifications(ctx context.Context) {
	if s.emailService == nil || s.emailTemplates == nil || s.userRepo == nil {
		return
	}
	for _, days := range []int{3, 1} {
		s.sendExpiringNotificationsForWindow(ctx, days)
	}
}

func (s *SubscriptionExpiryService) sendExpiringNotificationsForWindow(ctx context.Context, days int) {
	now := time.Now()
	upper := now.AddDate(0, 0, days)
	lower := upper.Add(-time.Hour)
	for page := 1; page <= 100; page++ {
		subs, result, err := s.userSubRepo.List(ctx, pagination.PaginationParams{Page: page, PageSize: 100}, nil, nil, SubscriptionStatusActive, "", "expires_at", "asc")
		if err != nil {
			log.Printf("[SubscriptionExpiry] List subscriptions for notification failed: %v", err)
			return
		}
		if len(subs) == 0 {
			return
		}
		for i := range subs {
			sub := subs[i]
			if sub.ExpiresAt.After(upper) {
				return
			}
			if sub.ExpiresAt.Before(lower) {
				continue
			}
			s.sendSubscriptionExpiringEmail(ctx, &sub, days, now)
		}
		if result == nil || page >= result.Pages {
			return
		}
	}
}

func (s *SubscriptionExpiryService) sendSubscriptionExpiringEmail(ctx context.Context, sub *UserSubscription, days int, now time.Time) {
	if sub == nil || sub.UserID <= 0 || sub.ID <= 0 {
		return
	}
	resourceID := "subscription:" + strconv.FormatInt(sub.ID, 10)
	window := "days:" + strconv.Itoa(days)
	if !s.emailTemplates.ShouldSendNotification(ctx, sub.UserID, NotificationCategorySubscriptionExp, resourceID, window, now) {
		return
	}
	user, err := s.userRepo.GetByID(ctx, sub.UserID)
	if err != nil || user == nil || strings.TrimSpace(user.Email) == "" {
		if err != nil {
			log.Printf("[SubscriptionExpiry] Load user for notification failed: user=%d err=%v", sub.UserID, err)
		}
		return
	}
	groupName := "Subscription"
	if sub.Group != nil && strings.TrimSpace(sub.Group.Name) != "" {
		groupName = strings.TrimSpace(sub.Group.Name)
	}
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}
	data := map[string]string{
		"SiteName":  siteName,
		"GroupName": groupName,
		"ExpiresAt": sub.ExpiresAt.Format("2006-01-02 15:04:05"),
		"DaysLeft":  strconv.Itoa(days),
	}
	if err := s.emailService.SendTemplatedEmail(ctx, user.Email, EmailTemplateSubscriptionExpiring, "zh", data); err != nil {
		log.Printf("[SubscriptionExpiry] Send subscription expiring email failed: user=%d subscription=%d err=%v", sub.UserID, sub.ID, err)
	}
}
