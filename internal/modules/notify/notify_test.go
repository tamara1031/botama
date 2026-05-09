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

// --- channelsConfigured ---

func TestChannelsConfigured(t *testing.T) {
	cases := []struct {
		name string
		ch   Channels
		want bool
	}{
		{"all empty", Channels{}, false},
		{"only info", Channels{Info: "ch"}, true},
		{"only warning", Channels{Warning: "ch"}, true},
		{"only critical", Channels{Critical: "ch"}, true},
		{"all set", Channels{Info: "a", Warning: "b", Critical: "c"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := channelsConfigured(tc.ch); got != tc.want {
				t.Errorf("channelsConfigured(%+v) = %v, want %v", tc.ch, got, tc.want)
			}
		})
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

func TestHealthz_JSONBody(t *testing.T) {
	n := &Notify{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	n.healthz(w, r)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: want application/json, got %q", ct)
	}
	if body := w.Body.String(); body != `{"status":"ok"}` {
		t.Errorf("body: want {\"status\":\"ok\"}, got %q", body)
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

// --- sendLevel ---

func TestSendLevel_NoChannel(t *testing.T) {
	n := newNotify("tok", Channels{}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify/info", nil)

	n.sendLevel(w, r, "info", "")

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestSendLevel_SendsToChannel(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", Channels{Info: "chan-info"}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify/info", bytes.NewBufferString(`{"content":"hello"}`))

	n.sendLevel(w, r, "info", "chan-info")

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

// --- New() route dispatch: data-driven integration tests ---

// newServer creates a full Notify server backed by a mock sender, using httptest.
func newServer(t *testing.T, token string, channels Channels, sender Sender) *httptest.Server {
	t.Helper()
	n := New(token, channels, ":0")
	n.sender = sender
	return httptest.NewServer(n.server.Handler)
}

func TestNew_RoutesAllLevels(t *testing.T) {
	cases := []struct {
		level   string
		path    string
		channel string
	}{
		{"info", "/notify/info", "chan-info"},
		{"warning", "/notify/warning", "chan-warn"},
		{"critical", "/notify/critical", "chan-crit"},
	}

	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			mock := &mockSender{}
			srv := newServer(t, "tok", Channels{
				Info:     "chan-info",
				Warning:  "chan-warn",
				Critical: "chan-crit",
			}, mock)
			defer srv.Close()

			body := bytes.NewBufferString(`{"content":"alert"}`)
			req, _ := http.NewRequest(http.MethodPost, srv.URL+tc.path, body)
			req.Header.Set("Authorization", "Bearer tok")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				t.Fatalf("want 204, got %d", resp.StatusCode)
			}
			if mock.sentTo != tc.channel {
				t.Fatalf("want sentTo=%q, got %q", tc.channel, mock.sentTo)
			}
			if mock.sentContent != "alert" {
				t.Fatalf("want sentContent=alert, got %q", mock.sentContent)
			}
		})
	}
}

func TestNew_UnauthorizedIsRejected(t *testing.T) {
	srv := newServer(t, "secret", Channels{Info: "ch"}, &mockSender{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/notify/info",
		bytes.NewBufferString(`{"content":"x"}`))
	req.Header.Set("Authorization", "Bearer wrong")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", resp.StatusCode)
	}
}

func TestNew_UnconfiguredChannelReturns404(t *testing.T) {
	srv := newServer(t, "tok", Channels{}, &mockSender{})
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/notify/warning",
		bytes.NewBufferString(`{"content":"x"}`))
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
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

	n.send(w, r, "info", "ch")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestSend_Success(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", Channels{}, mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"works!"}`))

	n.send(w, r, "info", "ch")

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if mock.sentContent != "works!" {
		t.Fatalf("want sentContent=works!, got %q", mock.sentContent)
	}
}
