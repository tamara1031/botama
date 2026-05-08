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

// statusResponseWriter wraps http.ResponseWriter to capture the written status code.
type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusResponseWriter) written() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

// requestLogger logs method, path, status, and duration for each request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusResponseWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		slog.Info("notify: request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.written(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	})
}

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
	auth := bearerAuth(token)
	mux.Handle("POST /notify/info", auth(http.HandlerFunc(n.handleInfo)))
	mux.Handle("POST /notify/warning", auth(http.HandlerFunc(n.handleWarning)))
	mux.Handle("POST /notify/critical", auth(http.HandlerFunc(n.handleCritical)))
	mux.HandleFunc("GET /healthz", n.healthz)
	n.server = &http.Server{
		Addr:              addr,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	return n
}

// bearerAuth returns middleware that enforces Bearer token authentication.
func bearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
}

func channelsConfigured(c Channels) bool {
	return c.Info != "" || c.Warning != "" || c.Critical != ""
}

func (n *Notify) Name() string { return "notify" }

func (n *Notify) Register(s *discordgo.Session) error {
	if n.token == "" {
		return fmt.Errorf("notify: API_TOKEN is required")
	}

	if !channelsConfigured(n.channels) {
		slog.Warn("notify: no notification channels configured; all /notify/* requests will return 404")
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

func (n *Notify) Shutdown(ctx context.Context) error {
	if err := n.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("notify: shutdown: %w", err)
	}
	return nil
}

func (n *Notify) healthz(w http.ResponseWriter, _ *http.Request) {
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
