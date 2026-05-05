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

type postBody struct {
	Content string `json:"content"`
}

type Notify struct {
	token            string
	defaultChannelID string
	server           *http.Server
	session          *discordgo.Session
}

func New(token, defaultChannelID, addr string) *Notify {
	n := &Notify{
		token:            token,
		defaultChannelID: defaultChannelID,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify/{channelID}", n.handleChannel)
	mux.HandleFunc("POST /notify", n.handleDefault)
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

	if _, err := n.session.ChannelMessageSend(channelID, body.Content); err != nil {
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

func (n *Notify) authorized(r *http.Request) bool {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(n.token)) == 1
}
