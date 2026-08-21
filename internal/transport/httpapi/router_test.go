package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/repository"
)

func testRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := repository.NewMemoryStore()
	clock := application.Clock(application.SystemClock)
	audit := application.AuditWriter{Repository: store, Clock: clock}
	campaigns := application.CampaignService{Campaigns: store, Catalog: store, Transactions: store, Audit: audit, Clock: clock}
	return NewRouter(API{Campaigns: campaigns, Ready: func(*gin.Context) error { return nil }}, []string{"http://localhost:5173"}, time.Second, t.TempDir())
}

func TestCreateCampaignRequiresLocalActor(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", strings.NewReader(`{"name":"处暑专题","season":"夏末","environment":"production","starts_at":"2026-08-22T08:00:00Z","ends_at":"2026-08-23T08:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), "AUTH_REQUIRED") {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateCampaignReturnsStableErrorEnvelope(t *testing.T) {
	router := testRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", strings.NewReader(`{"name":"处暑专题","season":"夏末","environment":"production","starts_at":"2026-08-22T08:00:00Z","ends_at":"2026-08-23T08:00:00Z"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Actor-ID", "operator")
	request.Header.Set("X-Actor-Role", "operator")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing idempotency key failure, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, field := range []string{"VALIDATION_FAILED", "field_errors", "request_id"} {
		if !strings.Contains(body, field) {
			t.Fatalf("response missing %s: %s", field, body)
		}
	}
}
