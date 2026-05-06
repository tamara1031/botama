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
func newNotify(token, defaultChannel string, sender Sender) *Notify {
	return &Notify{
		token:            token,
		defaultChannelID: defaultChannel,
		sender:           sender,
	}
}

// --- tokenAuth middleware ---

func TestTokenAuth_Unauthorized(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := tokenAuth("secret", inner)

	cases := []struct{ hdr string }{
		{""},
		{"wrong"},
		{"Bearer wrong"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		if tc.hdr != "" {
			req.Header.Set("Authorization", tc.hdr)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("header %q: got %d, want %d", tc.hdr, rr.Code, http.StatusUnauthorized)
		}
	}
}

func TestTokenAuth_Authorized(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := tokenAuth("secret", inner)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want %d", rr.Code, http.StatusOK)
	}
	if !called {
		t.Error("inner handler was not called")
	}
}

// --- handleDefault ---

func TestHandleDefault_NoDefaultChannel(t *testing.T) {
	n := newNotify("tok", "", nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify", nil)

	n.handleDefault(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestHandleDefault_SendsToDefaultChannel(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", "chan-123", mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":"hello"}`))

	n.handleDefault(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body)
	}
	if mock.sentTo != "chan-123" {
		t.Fatalf("want sentTo=chan-123, got %q", mock.sentTo)
	}
	if mock.sentContent != "hello" {
		t.Fatalf("want sentContent=hello, got %q", mock.sentContent)
	}
}

// --- handleChannel ---

func TestHandleChannel_SendsToPathChannel(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", "default-ch", mock)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify/{channelID}", n.handleChannel)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body := bytes.NewBufferString(`{"content":"hi there"}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/notify/ch-456", body)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
	if mock.sentTo != "ch-456" {
		t.Fatalf("want sentTo=ch-456, got %q", mock.sentTo)
	}
}

// --- send: body validation ---

func TestSend_BadJSON(t *testing.T) {
	n := newNotify("tok", "ch", &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))

	n.send(w, r, "ch")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestSend_EmptyContent(t *testing.T) {
	n := newNotify("tok", "ch", &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":""}`))

	n.send(w, r, "ch")

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

func TestSend_MissingContentField(t *testing.T) {
	n := newNotify("tok", "ch", &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))

	n.send(w, r, "ch")

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

// --- send: Discord error handling ---

func TestSend_ChannelNotFound(t *testing.T) {
	mock := &mockSender{err: restError(http.StatusNotFound)}
	n := newNotify("tok", "ch", mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"hi"}`))

	n.send(w, r, "unknown-ch")

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestSend_DiscordInternalError(t *testing.T) {
	mock := &mockSender{err: fmt.Errorf("discord exploded")}
	n := newNotify("tok", "ch", mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"hi"}`))

	n.send(w, r, "ch")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
}

func TestSend_Success(t *testing.T) {
	mock := &mockSender{}
	n := newNotify("tok", "ch", mock)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":"works!"}`))

	n.send(w, r, "ch")

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if mock.sentContent != "works!" {
		t.Fatalf("want sentContent=works!, got %q", mock.sentContent)
	}
}
