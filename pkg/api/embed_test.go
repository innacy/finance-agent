package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStaticFilesServed(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	srv.ServeStatic("../../web/dist")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /, got %d", w.Code)
	}
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty body for index.html")
	}
}

func TestAPIRoutesStillWork(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	srv.ServeStatic("../../web/dist")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/accounts", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("API should still work after static setup, got %d", w.Code)
	}
}
