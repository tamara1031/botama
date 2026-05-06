package notify

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const maxBodyBytes = 4 * 1024

// Sender is the subset of discordgo.Session used to post messages.
// *discordgo.Session satisfies this interface automatically.
type Sender interface {
	ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

type postBody struct {
	Content string `json:"content"`
}

type Notify struct {
	token            string
	defaultChannelID string
	server           *http.Server
	sender           Sender
}

func New(token, defaultChannelID, addr string) *Notify {
	n := &Notify{token: token, defaultChannelID: defaultChannelID}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify/{channelID}", n.handleChannel)
	mux.HandleFunc("POST /notify", n.handleDefault)
	n.server = &http.Server{
		Addr:              addr,
		Handler:           tokenAuth(token, mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	return n
}

// tokenAuth is an HTTP middleware that enforces Bearer token authentication.
func tokenAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) != 1 {
			slog.Warn("notify: unauthorized", "remote", r.RemoteAddr)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (n *Notify) Name() string { return "notify" }

func (n *Notify) Register(s *discordgo.Session) error {
	if n.token == "" {
		return fmt.Errorf("notify: API_TOKEN is required")
	}

	n.sender = s

	ln, err := net.Listen("tcp", n.server.Addr)
	if err != nil {
		return fmt.Errorf("notify: listen: %w", err)
	}
	go func() {
		slog.Info("notify: listening", "addr", n.server.Addr)
		if err := n.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			slog.Error("notify: server error", "error", err)
		}
	}()
	return nil
}

func (n *Notify) Unregister() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := n.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("notify: shutdown: %w", err)
	}
	return nil
}

// handleDefault sends to the configured default channel (backward compat).
func (n *Notify) handleDefault(w http.ResponseWriter, r *http.Request) {
	if n.defaultChannelID == "" {
		http.Error(w, "no default channel configured", http.StatusNotFound)
		return
	}
	n.send(w, r, n.defaultChannelID)
}

// handleChannel sends to the channel ID supplied in the path.
func (n *Notify) handleChannel(w http.ResponseWriter, r *http.Request) {
	n.send(w, r, r.PathValue("channelID"))
}

func (n *Notify) send(w http.ResponseWriter, r *http.Request, channelID string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var body postBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Content == "" {
		http.Error(w, "content required", http.StatusUnprocessableEntity)
		return
	}

	if _, err := n.sender.ChannelMessageSend(channelID, body.Content); err != nil {
		var restErr *discordgo.RESTError
		if errors.As(err, &restErr) && restErr.Response.StatusCode == http.StatusNotFound {
			slog.Warn("notify: channel not found", "channel", channelID)
			http.Error(w, "channel not found", http.StatusNotFound)
			return
		}
		slog.Error("notify: send failed", "error", err, "channel", channelID)
		http.Error(w, "failed to send", http.StatusInternalServerError)
		return
	}

	slog.Info("notify: sent", "channel", channelID, "remote", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}
