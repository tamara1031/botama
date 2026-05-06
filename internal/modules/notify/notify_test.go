package notify

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// fakeSender is a test double for messageSender.
type fakeSender struct {
	sentChannel string
	sentContent string
	err         error
}

func (f *fakeSender) ChannelMessageSend(channelID, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	f.sentChannel = channelID
	f.sentContent = content
	return nil, f.err
}

// notifyWithSender builds a Notify wired to a fake sender so tests skip
// the real TCP listener and Discord session.
func notifyWithSender(token, defaultChannelID string, sender messageSender) *Notify {
	n := New(token, defaultChannelID, ":0")
	n.sender = sender
	return n
}

func TestTokenAuth_unauthorized(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := tokenAuth("secret", inner)

	for _, hdr := range []string{"", "wrong", "Bearer wrong"} {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		if hdr != "" {
			req.Header.Set("Authorization", hdr)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("header %q: got %d, want %d", hdr, rr.Code, http.StatusUnauthorized)
		}
	}
}

func TestTokenAuth_authorized(t *testing.T) {
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

func TestHandleDefault_noChannel(t *testing.T) {
	n := notifyWithSender("tok", "", &fakeSender{})
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":"hi"}`))
	rr := httptest.NewRecorder()
	n.handleDefault(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestSend_emptyContent(t *testing.T) {
	n := notifyWithSender("tok", "ch1", &fakeSender{})
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":""}`))
	rr := httptest.NewRecorder()
	n.send(rr, req, "ch1")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d, want %d", rr.Code, http.StatusUnprocessableEntity)
	}
}

func TestSend_invalidJSON(t *testing.T) {
	n := notifyWithSender("tok", "ch1", &fakeSender{})
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`not-json`))
	rr := httptest.NewRecorder()
	n.send(rr, req, "ch1")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestSend_success(t *testing.T) {
	fs := &fakeSender{}
	n := notifyWithSender("tok", "ch1", fs)
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":"hello"}`))
	rr := httptest.NewRecorder()
	n.send(rr, req, "ch42")

	if rr.Code != http.StatusNoContent {
		t.Errorf("got %d, want %d", rr.Code, http.StatusNoContent)
	}
	if fs.sentChannel != "ch42" {
		t.Errorf("sentChannel = %q, want %q", fs.sentChannel, "ch42")
	}
	if fs.sentContent != "hello" {
		t.Errorf("sentContent = %q, want %q", fs.sentContent, "hello")
	}
}

func TestSend_discordNotFound(t *testing.T) {
	restErr := &discordgo.RESTError{
		Response: &http.Response{StatusCode: http.StatusNotFound},
		Message:  &discordgo.APIErrorMessage{},
	}
	fs := &fakeSender{err: restErr}
	n := notifyWithSender("tok", "", fs)
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":"hi"}`))
	rr := httptest.NewRecorder()
	n.send(rr, req, "bad-channel")

	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestSend_discordError(t *testing.T) {
	fs := &fakeSender{err: fmt.Errorf("network error")}
	n := notifyWithSender("tok", "", fs)
	req := httptest.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(`{"content":"hi"}`))
	rr := httptest.NewRecorder()
	n.send(rr, req, "ch1")

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
