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

type Channels struct {
	Info     string
	Warning  string
	Critical string
}

type Notify struct {
	token    string
	channels Channels
	server   *http.Server
	sender   Sender
}

func New(token string, channels Channels, addr string) *Notify {
	n := &Notify{
		token:    token,
		channels: channels,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", n.handleHealth)
	mux.HandleFunc("POST /notify/info", n.handleInfo)
	mux.HandleFunc("POST /notify/warning", n.handleWarning)
	mux.HandleFunc("POST /notify/critical", n.handleCritical)
	n.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	return n
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

func (n *Notify) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (n *Notify) handleInfo(w http.ResponseWriter, r *http.Request) {
	n.sendLevel(w, r, "info", n.channels.Info)
}

func (n *Notify) handleWarning(w http.ResponseWriter, r *http.Request) {
	n.sendLevel(w, r, "warning", n.channels.Warning)
}

func (n *Notify) handleCritical(w http.ResponseWriter, r *http.Request) {
	n.sendLevel(w, r, "critical", n.channels.Critical)
}

func (n *Notify) sendLevel(w http.ResponseWriter, r *http.Request, level, channelID string) {
	if channelID == "" {
		http.Error(w, "channel not configured", http.StatusNotFound)
		return
	}
	n.send(w, r, level, channelID)
}

func (n *Notify) send(w http.ResponseWriter, r *http.Request, level, channelID string) {
	if !n.authorized(r) {
		slog.Warn("notify: unauthorized", "remote", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

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
			slog.Warn("notify: channel not found", "level", level, "channel", channelID)
			http.Error(w, "channel not found", http.StatusNotFound)
			return
		}
		slog.Error("notify: send failed", "error", err, "level", level, "channel", channelID)
		http.Error(w, "failed to send", http.StatusInternalServerError)
		return
	}

	slog.Info("notify: sent", "level", level, "channel", channelID, "remote", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (n *Notify) authorized(r *http.Request) bool {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(n.token)) == 1
}
