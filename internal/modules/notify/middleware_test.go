package notify

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRequestID_Format(t *testing.T) {
	id := newRequestID()
	if len(id) != 16 {
		t.Fatalf("want 16-char hex ID, got %q (len %d)", id, len(id))
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("non-hex character %q in request ID %q", c, id)
		}
	}
}

func TestNewRequestID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := newRequestID()
		if ids[id] {
			t.Fatalf("duplicate request ID generated: %q", id)
		}
		ids[id] = true
	}
}

func TestRequestID_SetsHeader(t *testing.T) {
	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)

	id := w.Header().Get("X-Request-Id")
	if len(id) != 16 {
		t.Fatalf("want 16-char X-Request-Id header, got %q", id)
	}
}

func TestRequestID_InjectsContext(t *testing.T) {
	var captured string
	handler := requestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured, _ = r.Context().Value(requestIDKey).(string)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if len(captured) != 16 {
		t.Fatalf("want request ID in context, got %q", captured)
	}
}

func TestRequestLogger_PassesThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := requestLogger(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204 pass-through, got %d", w.Code)
	}
}

func TestNew_ResponseHasRequestID(t *testing.T) {
	srv := newServer(t, "tok", Channels{Info: "ch"}, &mockSender{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/healthz", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	id := resp.Header.Get("X-Request-Id")
	if len(id) != 16 {
		t.Fatalf("want X-Request-Id on response, got %q", id)
	}
}
