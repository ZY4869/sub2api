package setup

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
)

func TestDecideAdminBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalUsers int64
		adminUsers int64
		should     bool
		reason     string
	}{
		{
			name:       "empty database should create admin",
			totalUsers: 0,
			adminUsers: 0,
			should:     true,
			reason:     adminBootstrapReasonEmptyDatabase,
		},
		{
			name:       "admin exists should skip",
			totalUsers: 10,
			adminUsers: 1,
			should:     false,
			reason:     adminBootstrapReasonAdminExists,
		},
		{
			name:       "users exist without admin should skip",
			totalUsers: 5,
			adminUsers: 0,
			should:     false,
			reason:     adminBootstrapReasonUsersExistWithoutAdmin,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decideAdminBootstrap(tc.totalUsers, tc.adminUsers)
			if got.shouldCreate != tc.should {
				t.Fatalf("shouldCreate=%v, want %v", got.shouldCreate, tc.should)
			}
			if got.reason != tc.reason {
				t.Fatalf("reason=%q, want %q", got.reason, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "user_concurrency: 5") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
}

func TestRegisterRoutes_ExposesHealthEndpointDuringSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterRoutes(router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /health status=%d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Fatalf("GET /health body=%q, want status ok payload", w.Body.String())
	}
}

func TestSetupWindowOpen(t *testing.T) {
	t.Setenv("AUTO_SETUP", "false")
	t.Setenv("DATA_DIR", t.TempDir())
	if SetupWindowOpen() {
		t.Fatalf("SetupWindowOpen() should be false outside web setup server mode")
	}
}

func TestGetStatusReturnsClosedOutsideWebSetupMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AUTO_SETUP", "false")
	t.Setenv("DATA_DIR", t.TempDir())

	router := gin.New()
	RegisterRoutes(router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /setup/status status=%d, want %d", w.Code, http.StatusOK)
	}

	var payload struct {
		Data SetupStatus `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal setup status: %v", err)
	}
	if payload.Data.NeedsSetup {
		t.Fatalf("expected needs_setup=false outside web setup server mode")
	}
	if payload.Data.Step != "completed" {
		t.Fatalf("step=%q, want completed", payload.Data.Step)
	}
}

func TestGetStatusReturnsOpenInWebSetupMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AUTO_SETUP", "false")
	t.Setenv("DATA_DIR", t.TempDir())
	setWebSetupModeForTest(true)
	t.Cleanup(func() { setWebSetupModeForTest(false) })

	router := gin.New()
	RegisterRoutes(router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /setup/status status=%d, want %d", w.Code, http.StatusOK)
	}

	var payload struct {
		Data SetupStatus `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal setup status: %v", err)
	}
	if !payload.Data.NeedsSetup {
		t.Fatalf("expected needs_setup=true in web setup server mode")
	}
	if payload.Data.Step != "welcome" {
		t.Fatalf("step=%q, want welcome", payload.Data.Step)
	}
}

func TestProtectedSetupEndpointReachesHandlerInWebSetupMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("AUTO_SETUP", "false")
	t.Setenv("DATA_DIR", t.TempDir())
	setWebSetupModeForTest(true)
	t.Cleanup(func() { setWebSetupModeForTest(false) })

	router := gin.New()
	RegisterRoutes(router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/setup/test-db", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST /setup/test-db status=%d, want handler validation status %d", w.Code, http.StatusBadRequest)
	}
	if strings.Contains(w.Body.String(), setupClosedMessage) {
		t.Fatalf("POST /setup/test-db was blocked by setup guard, body=%q", w.Body.String())
	}
}

func TestProtectedSetupEndpointClosedWhenInstalled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)
	setWebSetupModeForTest(true)
	t.Cleanup(func() { setWebSetupModeForTest(false) })

	if err := os.WriteFile(GetInstallLockPath(), []byte("installed_at=2026-01-01T00:00:00Z\n"), 0400); err != nil {
		t.Fatalf("write install lock: %v", err)
	}

	router := gin.New()
	RegisterRoutes(router)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/setup/test-db", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("POST /setup/test-db status=%d, want %d", w.Code, http.StatusForbidden)
	}
	if !strings.Contains(w.Body.String(), setupClosedMessage) {
		t.Fatalf("POST /setup/test-db body=%q, want setup closed message", w.Body.String())
	}
}

func TestPostgresSetupDSNsUseMaintenanceDatabaseFirst(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "db.example.test",
		Port:     5432,
		User:     "postgres",
		Password: "secret",
		DBName:   "sub2api",
		SSLMode:  "disable",
	}

	maintenanceDSN, targetDSN := postgresSetupDSNs(cfg)

	if !strings.Contains(maintenanceDSN, "dbname=postgres") {
		t.Fatalf("maintenance DSN=%q, want dbname=postgres", maintenanceDSN)
	}
	if strings.Contains(maintenanceDSN, "dbname=sub2api") {
		t.Fatalf("maintenance DSN=%q should not point at target database", maintenanceDSN)
	}
	if !strings.Contains(targetDSN, "dbname=sub2api") {
		t.Fatalf("target DSN=%q, want dbname=sub2api", targetDSN)
	}
}

func TestDatabaseConnectionCreatesTargetThenVerifiesTarget(t *testing.T) {
	maintenanceDB, maintenanceMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock maintenance: %v", err)
	}
	targetDB, targetMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock target: %v", err)
	}

	cfg := &DatabaseConfig{
		Host:    "db.example.test",
		Port:    5432,
		User:    "postgres",
		DBName:  "sub2api",
		SSLMode: "disable",
	}
	maintenanceDSN, targetDSN := postgresSetupDSNs(cfg)
	openedDSNs := make([]string, 0, 2)
	originalOpenPostgres := openPostgres
	openPostgres = func(dsn string) (*sql.DB, error) {
		openedDSNs = append(openedDSNs, dsn)
		switch len(openedDSNs) {
		case 1:
			return maintenanceDB, nil
		case 2:
			return targetDB, nil
		default:
			t.Fatalf("unexpected postgres open call %d with dsn=%q", len(openedDSNs), dsn)
			return nil, nil
		}
	}
	t.Cleanup(func() { openPostgres = originalOpenPostgres })

	maintenanceMock.ExpectPing()
	maintenanceMock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)")).
		WithArgs("sub2api").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	maintenanceMock.ExpectExec(regexp.QuoteMeta("CREATE DATABASE sub2api")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	maintenanceMock.ExpectClose()
	targetMock.ExpectPing()
	targetMock.ExpectClose()

	if err := TestDatabaseConnection(cfg); err != nil {
		t.Fatalf("TestDatabaseConnection() error = %v", err)
	}
	if len(openedDSNs) != 2 {
		t.Fatalf("opened DSNs=%v, want maintenance then target", openedDSNs)
	}
	if openedDSNs[0] != maintenanceDSN {
		t.Fatalf("first opened DSN=%q, want maintenance %q", openedDSNs[0], maintenanceDSN)
	}
	if openedDSNs[1] != targetDSN {
		t.Fatalf("second opened DSN=%q, want target %q", openedDSNs[1], targetDSN)
	}
	if err := maintenanceMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("maintenance expectations: %v", err)
	}
	if err := targetMock.ExpectationsWereMet(); err != nil {
		t.Fatalf("target expectations: %v", err)
	}
}
