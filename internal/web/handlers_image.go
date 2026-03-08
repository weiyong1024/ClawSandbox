package web

import (
	"net/http"

	"github.com/weiyong1024/clawsandbox/internal/container"
)

// handleImageStatus reports whether the sandbox Docker image has been built.
func (s *Server) handleImageStatus(w http.ResponseWriter, r *http.Request) {
	exists, err := container.ImageExists(s.docker, s.config.ImageRef())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"image": s.config.ImageRef(),
			"built": exists,
		},
	})
}

// handleImageBuild triggers a Docker image build and streams progress via SSE.
func (s *Server) handleImageBuild(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	imageRef := s.config.ImageRef()
	pr, pw := newLineWriter()

	done := make(chan error, 1)
	go func() {
		done <- container.Build(s.docker, imageRef, pw)
		pw.Close()
	}()

	for line := range pr {
		writeSSE(w, "log", line)
		flusher.Flush()
	}

	if err := <-done; err != nil {
		writeSSE(w, "error", err.Error())
	} else {
		writeSSE(w, "done", "image built successfully")
	}
	flusher.Flush()
}

// writeSSE writes a single Server-Sent Event.
func writeSSE(w http.ResponseWriter, event, data string) {
	w.Write([]byte("event: " + event + "\ndata: " + data + "\n\n"))
}

// newLineWriter returns a channel that receives lines as they are written.
func newLineWriter() (<-chan string, *lineWriter) {
	ch := make(chan string, 64)
	return ch, &lineWriter{ch: ch}
}

type lineWriter struct {
	ch  chan string
	buf []byte
}

func (lw *lineWriter) Write(p []byte) (int, error) {
	lw.buf = append(lw.buf, p...)
	for {
		idx := -1
		for i, b := range lw.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := string(lw.buf[:idx])
		lw.buf = lw.buf[idx+1:]
		lw.ch <- line
	}
	return len(p), nil
}

func (lw *lineWriter) Close() error {
	if len(lw.buf) > 0 {
		lw.ch <- string(lw.buf)
		lw.buf = nil
	}
	close(lw.ch)
	return nil
}
