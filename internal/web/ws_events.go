package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// handleWSEvents forwards instance lifecycle events over a WebSocket.
func (s *Server) handleWSEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws events upgrade: %v", err)
		return
	}
	defer conn.Close()

	ch := s.events.Subscribe()
	defer s.events.Unsubscribe(ch)

	closeCh := make(chan struct{})
	go wsReadPump(conn, closeCh)

	for {
		select {
		case <-closeCh:
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			msg, _ := json.Marshal(evt)
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
