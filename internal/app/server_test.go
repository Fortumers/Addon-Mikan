package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestApp() *App {
	return &App{
		id:       AddonID,
		version:  "1.6.0",
		token:    "secret",
		settings: DefaultSettings(),
		ctx:      httptest.NewRequest(http.MethodGet, "/", nil).Context(),
	}
}

func TestInfoRequiresTokenAndNamesItself(t *testing.T) {
	a := newTestApp()
	h := a.routes()

	req := httptest.NewRequest(http.MethodGet, "/v2/info", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v2/info", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`"id":"fortumers-subs-bot"`, `"protocol":2`, `"/ui/index.html"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
}

func TestUIAndSettings(t *testing.T) {
	a := newTestApp()
	h := a.routes()

	req := httptest.NewRequest(http.MethodGet, "/ui/index.html", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "app.js") {
		t.Fatalf("page %d %s", rec.Code, rec.Body.String())
	}

	body := `{"method":"PUT","path":"/settings","admin":{"lang":"en"},"body":{"welcome":"Hello","autoreply":"Got it","buttons":[{"text":"Site","url":"https://example.com"}]}}`
	req = httptest.NewRequest(http.MethodPost, "/v2/ui/rpc", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rpc %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"lang":"en"`) || !strings.Contains(rec.Body.String(), "https://example.com") {
		t.Fatalf("body %s", rec.Body.String())
	}

	bad := `{"method":"PUT","path":"/settings","body":{"welcome":"Hi","buttons":[{"text":"X","url":"http://nope"}]}}`
	req = httptest.NewRequest(http.MethodPost, "/v2/ui/rpc", strings.NewReader(bad))
	req.Header.Set("Authorization", "Bearer secret")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestMarkupAndBroadcastGuard(t *testing.T) {
	s := DefaultSettings()
	if err := s.Normalize(); err != nil {
		t.Fatal(err)
	}
	kb := markup(s).(map[string]any)
	rows := kb["inline_keyboard"].([]any)
	btn := rows[0].([]any)[0].(map[string]any)
	if btn["callback_data"] != "b0" {
		t.Fatalf("callback %#v", btn)
	}
	a := newTestApp()
	if _, err := a.startBroadcast("hi"); err == nil {
		t.Fatal("broadcast with no chats should fail")
	}
}

func TestHealth(t *testing.T) {
	a := newTestApp()
	a.token = ""
	rec := httptest.NewRecorder()
	a.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/health", nil))
	b, _ := io.ReadAll(rec.Body)
	if rec.Code != 200 || string(b) != `{"ok":true}` {
		t.Fatalf("%d %s", rec.Code, b)
	}
}
