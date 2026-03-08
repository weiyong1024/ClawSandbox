package web

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/weiyong1024/clawsandbox/internal/container"
)

// handleWSLogs streams container logs over a WebSocket (follow mode).
func (s *Server) handleWSLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	store, err := s.loadStore()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	inst := store.Get(name)
	if inst == nil {
		http.Error(w, fmt.Sprintf("instance %s not found", name), http.StatusNotFound)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws logs upgrade: %v", err)
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		defer cancel()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	writer := &wsWriter{conn: conn}
	if err := container.LogsFollow(s.docker, inst.ContainerID, ctx, writer); err != nil {
		if ctx.Err() == nil {
			log.Printf("ws logs %s: %v", name, err)
		}
	}
}

// wsWriter sends each Write call as a WebSocket text message.
type wsWriter struct {
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}
