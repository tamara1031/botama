package notify

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const maxBodyBytes = 4 * 1024

type postBody struct {
	Content string `json:"content"`
}

type Notify struct {
	token    string
	channels map[string]string // channel name → Discord channel ID
	server   *http.Server
	session  *discordgo.Session
}

func New(token string, channels map[string]string, addr string) *Notify {
	n := &Notify{
		token:    token,
		channels: channels,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify/{channel}", n.handle)
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
	if len(n.channels) == 0 {
		return fmt.Errorf("notify: at least one channel must be configured via NOTIFY_CHANNELS or NOTIFICATION_CHANNEL_ID")
	}

	n.session = s

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

func (n *Notify) handle(w http.ResponseWriter, r *http.Request) {
	if !n.authorized(r) {
		slog.Warn("notify: unauthorized", "remote", r.RemoteAddr)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	channelName := r.PathValue("channel")
	channelID, ok := n.channels[channelName]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown channel: %q", channelName), http.StatusNotFound)
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

	if _, err := n.session.ChannelMessageSend(channelID, body.Content); err != nil {
		slog.Error("notify: send failed", "error", err, "channel", channelName)
		http.Error(w, "failed to send", http.StatusInternalServerError)
		return
	}

	slog.Info("notify: sent", "channel", channelName, "remote", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (n *Notify) authorized(r *http.Request) bool {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(n.token)) == 1
}
