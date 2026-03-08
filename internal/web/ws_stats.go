package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"github.com/weiyong1024/clawsandbox/internal/container"
)

// wsUpgrader is shared by all WebSocket handlers.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Only allow same-host connections (localhost → localhost).
		return r.Header.Get("Origin") == "" ||
			r.Host == r.Header.Get("Origin") ||
			r.Header.Get("Origin") == "http://"+r.Host ||
			r.Header.Get("Origin") == "https://"+r.Host
	},
}

type instanceStats struct {
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryUsage int64   `json:"memory_usage"`
	MemoryLimit int64   `json:"memory_limit"`
}

// handleWSStats pushes CPU/memory stats for all running instances every 2 seconds.
func (s *Server) handleWSStats(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws stats upgrade: %v", err)
		return
	}
	defer conn.Close()

	closeCh := make(chan struct{})
	go wsReadPump(conn, closeCh)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-closeCh:
			return
		case <-ticker.C:
			store, err := s.loadStore()
			if err != nil {
				log.Printf("ws stats load store: %v", err)
				continue
			}

			instances := store.Snapshot()
			results := make([]instanceStats, 0, len(instances))
			for _, inst := range instances {
				status, _, _ := container.Status(s.docker, inst.ContainerID)
				if status != "running" {
					continue
				}
				st, err := container.Stats(s.docker, inst.ContainerID)
				if err != nil {
					log.Printf("ws stats %s: %v", inst.Name, err)
					continue
				}
				results = append(results, instanceStats{
					Name:        inst.Name,
					CPUPercent:  st.CPUPercent,
					MemoryUsage: st.MemoryUsage,
					MemoryLimit: st.MemoryLimit,
				})
			}

			msg, _ := json.Marshal(map[string]any{"instances": results})
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

// wsReadPump drains incoming messages and closes closeCh when the peer disconnects.
func wsReadPump(conn *websocket.Conn, closeCh chan struct{}) {
	defer close(closeCh)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
