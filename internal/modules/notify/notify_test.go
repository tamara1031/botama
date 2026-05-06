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

// restError builds a *discordgo.RESTError with the given HTTP status code so
// tests can exercise the "channel not found" path without a live connection.
func restError(status int) *discordgo.RESTError {
	return &discordgo.RESTError{
		Response: &http.Response{StatusCode: status},
		Message:  &discordgo.APIErrorMessage{},
	}
}

func newNotify(token, defaultChannel string, sender Sender) *Notify {
	n := &Notify{
		token:            token,
		defaultChannelID: defaultChannel,
		sender:           sender,
	}
	return n
}

// --- authorized ---

func TestAuthorized(t *testing.T) {
	n := newNotify("secret", "", nil)

	t.Run("valid bearer token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set("Authorization", "Bearer secret")
		if !n.authorized(r) {
			t.Fatal("expected authorized")
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set("Authorization", "Bearer wrong")
		if n.authorized(r) {
			t.Fatal("expected unauthorized")
		}
	})

	t.Run("missing header", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		if n.authorized(r) {
			t.Fatal("expected unauthorized")
		}
	})
}

// --- handleDefault ---

func TestHandleDefault_NoDefaultChannel(t *testing.T) {
	n := newNotify("tok", "", nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify", nil)
	r.Header.Set("Authorization", "Bearer tok")

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
	r.Header.Set("Authorization", "Bearer tok")

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
	req.Header.Set("Authorization", "Bearer tok")
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

// --- send: auth ---

func TestSend_Unauthorized(t *testing.T) {
	n := newNotify("tok", "ch", nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":"x"}`))
	// No Authorization header

	n.send(w, r, "ch")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

// --- send: body validation ---

func TestSend_BadJSON(t *testing.T) {
	n := newNotify("tok", "ch", &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-json"))
	r.Header.Set("Authorization", "Bearer tok")

	n.send(w, r, "ch")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestSend_EmptyContent(t *testing.T) {
	n := newNotify("tok", "ch", &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"content":""}`))
	r.Header.Set("Authorization", "Bearer tok")

	n.send(w, r, "ch")

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

func TestSend_MissingContentField(t *testing.T) {
	n := newNotify("tok", "ch", &mockSender{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{}`))
	r.Header.Set("Authorization", "Bearer tok")

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
	r.Header.Set("Authorization", "Bearer tok")

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
	r.Header.Set("Authorization", "Bearer tok")

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
	r.Header.Set("Authorization", "Bearer tok")

	n.send(w, r, "ch")

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d", w.Code)
	}
	if mock.sentContent != "works!" {
		t.Fatalf("want sentContent=works!, got %q", mock.sentContent)
	}
}
