package notify

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// mockSender records calls and returns configurable errors.
type mockSender struct {
	sentTo      string
	sentContent string
	err         error
}

func (m *mockSender) ChannelMessageSend(channelID, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.sentTo = channelID
	m.sentContent = content
	if m.err != nil {
		return nil, m.err
	}
	return &discordgo.Message{ID: "fake-id"}, nil
}

// restError builds a *discordgo.RESTError with the given HTTP status code.
func restError(status int) *discordgo.RESTError {
	return &discordgo.RESTError{
		Response: &http.Response{StatusCode: status},
		Message:  &discordgo.APIErrorMessage{},
	}
}

// newNotify constructs a Notify with a fake sender, skipping the TCP listener.
func newNotify(token string, channels Channels, sender Sender) *Notify {
	return &Notify{
		token:    token,
		channels: channels,
		sender:   sender,
	}
}

// --- healthz ---

func TestHealthz(t *testing.T) {
	n := &Notify{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	n.healthz(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

// --- bearerAuth middleware ---

func TestBearerAuth(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := bearerAuth("secret")(inner)

	t.Run("valid bearer token", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set("Authorization", "Bearer secret")
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", w.Code)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set("Authorization", "Bearer wrong")
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", w.Code)
		}
	})
}

// --- handleInfo ---

func TestHandleInfo_NoChannel(t *testing.T) {
	n := newNotify("tok", Channels{}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify/info", nil)

	n.handleInfo(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleInfo_SendsToInfoChannel(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", Channels{Info: "chan-info"}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify/info", bytes.NewBufferString(`{"content":"hello"}`))

	n.handleInfo(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body)
	}
	if mock.sentTo != "chan-info" {
		t.Fatalf("want sentTo=chan-info, got %q", mock.sentTo)
	}
	if mock.sentContent != "hello" {
		t.Fatalf("want sentContent=hello, got %q", mock.sentContent)
	}
}

// --- handleWarning ---

func TestHandleWarning_SendsToWarningChannel(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", Channels{Warning: "chan-warn"}, mock)

	mux := http.NewServeMux()
	mux.Handle("POST /notify/warning", bearerAuth("tok")(http.HandlerFunc(n.handleWarning)))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := bytes.NewBufferString(`{"content":"high cpu"}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/notify/warning", body)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
	if mock.sentTo != "chan-warn" {
		t.Fatalf("want sentTo=chan-warn, got %q", mock.sentTo)
	}
}

// --- handleCritical ---

func TestHandleCritical_SendsToCriticalChannel(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", Channels{Critical: "chan-crit"}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify/critical", bytes.NewBufferString(`{"content":"down"}`))

	n.handleCritical(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body)
	}
	if mock.sentTo != "chan-crit" {
		t.Fatalf("want sentTo=chan-crit, got %q", mock.sentTo)
	}
}

// --- send: body validation ---

func TestSend_BadJSON(t *testing.T) {
	n := newNotify("tok", Channels{}, &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))

	n.send(w, r, "info", "ch")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestSend_EmptyContent(t *testing.T) {
	n := newNotify("tok", Channels{}, &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":""}`))

	n.send(w, r, "info", "ch")

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

func TestSend_MissingContentField(t *testing.T) {
	n := newNotify("tok", Channels{}, &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))

	n.send(w, r, "info", "ch")

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

// --- send: Discord error handling ---

func TestSend_ChannelNotFound(t *testing.T) {
	mock := &mockSender{err: restError(http.StatusNotFound)}
	n := newNotify("tok", Channels{}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"hi"}`))

	n.send(w, r, "info", "unknown-ch")

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestSend_DiscordInternalError(t *testing.T) {
	mock := &mockSender{err: fmt.Errorf("discord exploded")}
	n := newNotify("tok", Channels{}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"hi"}`))

	n.send(w, r, "critical", "ch")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestSend_Success(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", Channels{}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"works!"}`))

	n.send(w, r, "warning", "ch")

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if mock.sentContent != "works!" {
		t.Fatalf("want sentContent=works!, got %q", mock.sentContent)
	}
}
