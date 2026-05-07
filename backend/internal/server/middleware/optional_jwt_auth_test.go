package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type optionalJWTUserRepoStub struct {
	user *service.User
}

func (s *optionalJWTUserRepoStub) Create(ctx context.Context, user *service.User) error {
	panic("unexpected Create call")
}

func (s *optionalJWTUserRepoStub) GetByID(ctx context.Context, id int64) (*service.User, error) {
	if s.user != nil && s.user.ID == id {
		clone := *s.user
		return &clone, nil
	}
	return nil, service.ErrUserNotFound
}

func (s *optionalJWTUserRepoStub) GetByEmail(ctx context.Context, email string) (*service.User, error) {
	panic("unexpected GetByEmail call")
}

func (s *optionalJWTUserRepoStub) GetFirstAdmin(ctx context.Context) (*service.User, error) {
	panic("unexpected GetFirstAdmin call")
}

func (s *optionalJWTUserRepoStub) Update(ctx context.Context, user *service.User) error {
	panic("unexpected Update call")
}

func (s *optionalJWTUserRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *optionalJWTUserRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *optionalJWTUserRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters service.UserListFilters) ([]service.User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *optionalJWTUserRepoStub) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *optionalJWTUserRepoStub) DeductBalance(ctx context.Context, id int64, amount float64) error {
	panic("unexpected DeductBalance call")
}

func (s *optionalJWTUserRepoStub) UpdateConcurrency(ctx context.Context, id int64, amount int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *optionalJWTUserRepoStub) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *optionalJWTUserRepoStub) RemoveGroupFromAllowedGroups(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *optionalJWTUserRepoStub) AddGroupToAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *optionalJWTUserRepoStub) RemoveGroupFromUserAllowedGroups(ctx context.Context, userID int64, groupID int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *optionalJWTUserRepoStub) UpdateTotpSecret(ctx context.Context, userID int64, encryptedSecret *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *optionalJWTUserRepoStub) EnableTotp(ctx context.Context, userID int64) error {
	panic("unexpected EnableTotp call")
}

func (s *optionalJWTUserRepoStub) DisableTotp(ctx context.Context, userID int64) error {
	panic("unexpected DisableTotp call")
}

func TestOptionalJWTAuthMiddleware_PopulatesContextForValidBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &optionalJWTUserRepoStub{
		user: &service.User{
			ID:           7,
			Email:        "admin@example.com",
			Role:         service.RoleAdmin,
			Concurrency:  9,
			Status:       service.StatusActive,
			TokenVersion: 2,
		},
	}
	authService := service.NewAuthService(nil, userRepo, nil, nil, &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			ExpireHour: 24,
		},
	}, nil, nil, nil, nil, nil, nil, nil)
	token, err := authService.GenerateToken(userRepo.user)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	userService := service.NewUserService(userRepo, nil, nil)

	r := gin.New()
	r.Use(NewOptionalJWTAuthMiddleware(authService, userService))
	r.GET("/probe", func(c *gin.Context) {
		subject, ok := GetAuthSubjectFromContext(c)
		if !ok {
			c.String(http.StatusUnauthorized, "missing subject")
			return
		}
		role, _ := GetUserRoleFromContext(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":     subject.UserID,
			"concurrency": subject.Concurrency,
			"role":        role,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
}

func TestOptionalJWTAuthMiddleware_IgnoresMissingOrInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &optionalJWTUserRepoStub{}
	authService := service.NewAuthService(nil, userRepo, nil, nil, &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			ExpireHour: 24,
		},
	}, nil, nil, nil, nil, nil, nil, nil)
	userService := service.NewUserService(userRepo, nil, nil)

	cases := []struct {
		name   string
		header string
	}{
		{name: "missing"},
		{name: "wrong_scheme", header: "Basic test"},
		{name: "invalid_token", header: "Bearer not-a-jwt"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(NewOptionalJWTAuthMiddleware(authService, userService))
			r.GET("/probe", func(c *gin.Context) {
				if _, ok := GetAuthSubjectFromContext(c); ok {
					c.String(http.StatusInternalServerError, "unexpected subject")
					return
				}
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want 204 body=%s", w.Code, w.Body.String())
			}
		})
	}
}
