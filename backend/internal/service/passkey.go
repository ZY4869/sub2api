package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var (
	ErrPasskeysDisabled = infraerrors.Forbidden("PASSKEYS_DISABLED", "passkeys are disabled")
	ErrPasskeyNotFound  = infraerrors.NotFound("PASSKEY_NOT_FOUND", "passkey not found")
	ErrPasskeyExists    = infraerrors.Conflict("PASSKEY_EXISTS", "passkey already exists")
	ErrPasskeySession   = infraerrors.BadRequest("PASSKEY_SESSION_INVALID", "passkey session is invalid or expired")
	ErrPasskeyVerify    = infraerrors.Unauthorized("PASSKEY_VERIFY_FAILED", "passkey verification failed")
)

const (
	passkeySessionTTL      = 5 * time.Minute
	passkeyDefaultName     = "Passkey"
	passkeyUserHandleBytes = 32
)

type PasskeyCredential struct {
	ID              int64
	UserID          int64
	CredentialID    []byte
	Name            string
	Credential      webauthn.Credential
	LastUsedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CredentialIDB64 string
}

type PasskeyRepository interface {
	GetOrCreateUserHandle(ctx context.Context, userID int64) ([]byte, error)
	GetUserIDByHandle(ctx context.Context, handle []byte) (int64, error)
	ListByUser(ctx context.Context, userID int64) ([]PasskeyCredential, error)
	GetByIDForUser(ctx context.Context, id, userID int64) (*PasskeyCredential, error)
	GetByCredentialID(ctx context.Context, credentialID []byte) (*PasskeyCredential, error)
	CreateCredential(ctx context.Context, credential *PasskeyCredential) error
	UpdateCredential(ctx context.Context, credentialID []byte, credential webauthn.Credential, lastUsedAt time.Time) error
	UpdateName(ctx context.Context, id, userID int64, name string) error
	Delete(ctx context.Context, id, userID int64) error
}

type PasskeySessionStore interface {
	Store(ctx context.Context, purpose, sessionID string, data webauthn.SessionData, ttl time.Duration) error
	Take(ctx context.Context, purpose, sessionID string) (*webauthn.SessionData, error)
}

type PasskeyRegistrationBegin struct {
	SessionID string                       `json:"session_id"`
	Options   *protocol.CredentialCreation `json:"options"`
}

type PasskeyLoginBegin struct {
	SessionID string                        `json:"session_id"`
	Options   *protocol.CredentialAssertion `json:"options"`
}

type PasskeyLoginResult struct {
	TokenPair *TokenPair
	User      *User
}

type PasskeyService struct {
	cfg          *config.Config
	auth         *AuthService
	repo         PasskeyRepository
	sessionStore PasskeySessionStore
}

func NewPasskeyService(cfg *config.Config, authService *AuthService, repo PasskeyRepository, sessionStore PasskeySessionStore) *PasskeyService {
	return &PasskeyService{
		cfg:          cfg,
		auth:         authService,
		repo:         repo,
		sessionStore: sessionStore,
	}
}

func (s *PasskeyService) Enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.WebAuthn.Enabled
}

func (s *PasskeyService) List(ctx context.Context, userID int64) ([]PasskeyCredential, error) {
	if !s.Enabled() {
		return []PasskeyCredential{}, nil
	}
	if s.repo == nil {
		return nil, ErrPasskeysDisabled
	}
	items, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list passkeys: %w", err)
	}
	return items, nil
}

func (s *PasskeyService) BeginRegistration(ctx context.Context, userID int64, password string) (*PasskeyRegistrationBegin, error) {
	wa, err := s.webAuthn()
	if err != nil {
		return nil, err
	}
	if s.auth == nil || s.auth.userRepo == nil || s.repo == nil || s.sessionStore == nil {
		return nil, ErrPasskeysDisabled
	}
	user, err := s.auth.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.CheckPassword(password) {
		return nil, ErrPasswordIncorrect
	}
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load passkeys: %w", err)
	}
	handle, err := s.repo.GetOrCreateUserHandle(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get passkey handle: %w", err)
	}
	waUser := newPasskeyWebAuthnUser(user, handle, creds)
	exclusions := make([]protocol.CredentialDescriptor, 0, len(creds))
	for i := range creds {
		exclusions = append(exclusions, creds[i].Credential.Descriptor())
	}
	requireResident := true
	options, session, err := wa.BeginRegistration(
		waUser,
		webauthn.WithExclusions(exclusions),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: &requireResident,
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationPreferred,
		}),
	)
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] begin registration failed: %v", err)
		return nil, ErrPasskeyVerify
	}
	sessionID, err := randomPasskeySessionID()
	if err != nil {
		return nil, fmt.Errorf("generate passkey session: %w", err)
	}
	if err := s.sessionStore.Store(ctx, "register", sessionID, *session, passkeySessionTTL); err != nil {
		return nil, fmt.Errorf("store passkey session: %w", err)
	}
	return &PasskeyRegistrationBegin{SessionID: sessionID, Options: options}, nil
}

func (s *PasskeyService) FinishRegistration(ctx context.Context, userID int64, sessionID, name string, response []byte) (*PasskeyCredential, error) {
	wa, err := s.webAuthn()
	if err != nil {
		return nil, err
	}
	if s.auth == nil || s.auth.userRepo == nil || s.repo == nil || s.sessionStore == nil {
		return nil, ErrPasskeysDisabled
	}
	user, err := s.auth.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	handle, err := s.repo.GetOrCreateUserHandle(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get passkey handle: %w", err)
	}
	session, err := s.sessionStore.Take(ctx, "register", sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil || !bytes.Equal(session.UserID, handle) {
		return nil, ErrPasskeySession
	}
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load passkeys: %w", err)
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] parse registration response failed: %v", err)
		return nil, ErrPasskeyVerify
	}
	credential, err := wa.CreateCredential(newPasskeyWebAuthnUser(user, handle, creds), *session, parsed)
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] finish registration failed: %v", err)
		return nil, ErrPasskeyVerify
	}
	item := &PasskeyCredential{
		UserID:       userID,
		CredentialID: credential.ID,
		Name:         normalizePasskeyName(name),
		Credential:   *credential,
	}
	if err := s.repo.CreateCredential(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *PasskeyService) BeginLogin(ctx context.Context) (*PasskeyLoginBegin, error) {
	wa, err := s.webAuthn()
	if err != nil {
		return nil, err
	}
	if s.repo == nil || s.sessionStore == nil {
		return nil, ErrPasskeysDisabled
	}
	options, session, err := wa.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] begin login failed: %v", err)
		return nil, ErrPasskeyVerify
	}
	sessionID, err := randomPasskeySessionID()
	if err != nil {
		return nil, fmt.Errorf("generate passkey session: %w", err)
	}
	if err := s.sessionStore.Store(ctx, "login", sessionID, *session, passkeySessionTTL); err != nil {
		return nil, fmt.Errorf("store passkey login session: %w", err)
	}
	return &PasskeyLoginBegin{SessionID: sessionID, Options: options}, nil
}

func (s *PasskeyService) FinishLogin(ctx context.Context, sessionID string, response []byte) (*PasskeyLoginResult, error) {
	wa, err := s.webAuthn()
	if err != nil {
		return nil, err
	}
	if s.auth == nil || s.auth.userRepo == nil || s.repo == nil || s.sessionStore == nil {
		return nil, ErrPasskeysDisabled
	}
	session, err := s.sessionStore.Take(ctx, "login", sessionID)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] parse login response failed: %v", err)
		return nil, ErrPasskeyVerify
	}
	var selected *passkeyWebAuthnUser
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		waUser, lookupErr := s.lookupWebAuthnUser(ctx, rawID, userHandle)
		if lookupErr != nil {
			return nil, lookupErr
		}
		selected = waUser
		return waUser, nil
	}
	_, credential, err := wa.ValidatePasskeyLogin(handler, *session, parsed)
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] finish login failed: %v", err)
		return nil, ErrPasskeyVerify
	}
	if selected == nil || selected.user == nil {
		return nil, ErrPasskeyVerify
	}
	if !selected.user.IsActive() {
		return nil, ErrUserNotActive
	}
	if err := s.repo.UpdateCredential(ctx, credential.ID, *credential, time.Now()); err != nil {
		return nil, fmt.Errorf("update passkey credential: %w", err)
	}
	tokenPair, err := s.auth.GenerateTokenPair(ctx, selected.user, "")
	if err != nil {
		return nil, fmt.Errorf("generate token pair: %w", err)
	}
	return &PasskeyLoginResult{TokenPair: tokenPair, User: selected.user}, nil
}

func (s *PasskeyService) Rename(ctx context.Context, userID, id int64, name string) error {
	if !s.Enabled() {
		return ErrPasskeysDisabled
	}
	if s.repo == nil {
		return ErrPasskeysDisabled
	}
	name = normalizePasskeyName(name)
	return s.repo.UpdateName(ctx, id, userID, name)
}

func (s *PasskeyService) Delete(ctx context.Context, userID, id int64, password string) error {
	if !s.Enabled() {
		return ErrPasskeysDisabled
	}
	if s.auth == nil || s.auth.userRepo == nil || s.repo == nil {
		return ErrPasskeysDisabled
	}
	user, err := s.auth.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.CheckPassword(password) {
		return ErrPasswordIncorrect
	}
	return s.repo.Delete(ctx, id, userID)
}

func (s *PasskeyService) webAuthn() (*webauthn.WebAuthn, error) {
	if !s.Enabled() {
		return nil, ErrPasskeysDisabled
	}
	rpName := strings.TrimSpace(s.cfg.WebAuthn.RPDisplayName)
	if rpName == "" {
		rpName = "Sub2API"
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          strings.TrimSpace(s.cfg.WebAuthn.RPID),
		RPDisplayName: rpName,
		RPOrigins:     s.cfg.WebAuthn.RPOrigins,
	})
	if err != nil {
		logger.LegacyPrintf("service.passkey", "[Passkey] webauthn config invalid: %v", err)
		return nil, ErrPasskeysDisabled
	}
	return wa, nil
}

func (s *PasskeyService) lookupWebAuthnUser(ctx context.Context, rawID, userHandle []byte) (*passkeyWebAuthnUser, error) {
	var userID int64
	if len(userHandle) > 0 {
		id, err := s.repo.GetUserIDByHandle(ctx, userHandle)
		if err != nil {
			return nil, err
		}
		userID = id
	} else if len(rawID) > 0 {
		credential, err := s.repo.GetByCredentialID(ctx, rawID)
		if err != nil {
			return nil, err
		}
		userID = credential.UserID
	} else {
		return nil, ErrPasskeyVerify
	}
	user, err := s.auth.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	handle, err := s.repo.GetOrCreateUserHandle(ctx, userID)
	if err != nil {
		return nil, err
	}
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return newPasskeyWebAuthnUser(user, handle, creds), nil
}

type passkeyWebAuthnUser struct {
	user        *User
	handle      []byte
	credentials []webauthn.Credential
}

func newPasskeyWebAuthnUser(user *User, handle []byte, records []PasskeyCredential) *passkeyWebAuthnUser {
	credentials := make([]webauthn.Credential, 0, len(records))
	for i := range records {
		credentials = append(credentials, records[i].Credential)
	}
	return &passkeyWebAuthnUser{user: user, handle: handle, credentials: credentials}
}

func (u *passkeyWebAuthnUser) WebAuthnID() []byte {
	if u == nil {
		return nil
	}
	return u.handle
}

func (u *passkeyWebAuthnUser) WebAuthnName() string {
	if u == nil || u.user == nil {
		return ""
	}
	return u.user.Email
}

func (u *passkeyWebAuthnUser) WebAuthnDisplayName() string {
	if u == nil || u.user == nil {
		return ""
	}
	if strings.TrimSpace(u.user.Username) != "" {
		return strings.TrimSpace(u.user.Username)
	}
	return u.user.Email
}

func (u *passkeyWebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	if u == nil {
		return nil
	}
	return u.credentials
}

func MarshalPasskeyCredentialData(credential webauthn.Credential) (map[string]any, error) {
	raw, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func UnmarshalPasskeyCredentialData(data map[string]any) (webauthn.Credential, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return webauthn.Credential{}, err
	}
	var credential webauthn.Credential
	if err := json.Unmarshal(raw, &credential); err != nil {
		return webauthn.Credential{}, err
	}
	return credential, nil
}

func NewPasskeyUserHandle() ([]byte, error) {
	out := make([]byte, passkeyUserHandleBytes)
	if _, err := rand.Read(out); err != nil {
		return nil, err
	}
	return out, nil
}

func randomPasskeySessionID() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func normalizePasskeyName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return passkeyDefaultName
	}
	runes := []rune(name)
	if len(runes) > 80 {
		name = string(runes[:80])
	}
	return name
}
