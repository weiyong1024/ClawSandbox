package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// handleConsoleProxy reverse-proxies requests to an instance's OpenClaw Gateway
// web UI. This allows Dashboard users to access the control panel without
// direct access to the container's loopback port.
//
// Route: /console/{name}/* → http://127.0.0.1:{gateway_port}/*
func (s *Server) handleConsoleProxy(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	store, err := s.loadStore()
	if err != nil {
		http.Error(w, "failed to load state", http.StatusInternalServerError)
		return
	}

	inst := store.Get(name)
	if inst == nil {
		http.Error(w, fmt.Sprintf("instance %q not found", name), http.StatusNotFound)
		return
	}
	if inst.Status != "running" {
		http.Error(w, fmt.Sprintf("instance %q is not running", name), http.StatusServiceUnavailable)
		return
	}

	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", inst.Ports.Gateway))

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Strip the /console/{name} prefix from the path
			prefix := fmt.Sprintf("/console/%s", name)
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.URL.RawPath = ""
		},
	}

	proxy.ServeHTTP(w, r)
}
