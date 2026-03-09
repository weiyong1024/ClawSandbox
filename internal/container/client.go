package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

func NewClient() (*docker.Client, error) {
	if hasExplicitDockerEnv() {
		cli, err := docker.NewClientFromEnv()
		if err != nil {
			return nil, fmt.Errorf("connecting to Docker from environment: %w", err)
		}
		if err := cli.Ping(); err != nil {
			return nil, fmt.Errorf("connecting to Docker using DOCKER_HOST: %w\nThis is usually a local Docker socket/context issue, not a network issue.", err)
		}
		return cli, nil
	}

	contextHost, err := dockerContextHostCandidate()
	if err != nil {
		return nil, err
	}

	candidates := dockerHostCandidates(contextHost)
	tried := make([]string, 0, len(candidates))
	for _, host := range candidates {
		cli, err := docker.NewClient(host)
		if err != nil {
			tried = append(tried, fmt.Sprintf("%s (%v)", host, err))
			continue
		}
		if err := cli.Ping(); err == nil {
			return cli, nil
		} else {
			tried = append(tried, fmt.Sprintf("%s (%v)", host, err))
		}
	}

	return nil, fmt.Errorf("connecting to Docker failed.\nTried: %s\nStart Docker Desktop and make sure the current Docker socket/context is reachable. This is usually a local Docker setup issue, not a network issue.",
		strings.Join(tried, "; "))
}

func hasExplicitDockerEnv() bool {
	return os.Getenv("DOCKER_HOST") != "" ||
		os.Getenv("DOCKER_TLS_VERIFY") != "" ||
		os.Getenv("DOCKER_CERT_PATH") != "" ||
		os.Getenv("DOCKER_API_VERSION") != ""
}

func dockerHostCandidates(contextHost string) []string {
	hosts := make([]string, 0, 4)
	if contextHost != "" {
		hosts = append(hosts, contextHost)
	}

	if home, err := os.UserHomeDir(); err == nil {
		hosts = appendIfSocketExists(hosts, filepath.Join(home, ".docker", "run", "docker.sock"))
		hosts = appendIfSocketExists(hosts, filepath.Join(home, ".colima", "default", "docker.sock"))
	}

	hosts = appendIfSocketExists(hosts, "/var/run/docker.sock")
	if len(hosts) == 0 {
		hosts = append(hosts, "unix:///var/run/docker.sock")
	}
	return uniqueStrings(hosts)
}

func appendIfSocketExists(hosts []string, path string) []string {
	if _, err := os.Stat(path); err == nil {
		return append(hosts, "unix://"+path)
	}
	return hosts
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

type dockerContextInfo struct {
	Name           string
	Host           string
	SkipTLSVerify  bool
	HasTLSMaterial bool
}

func dockerContextHostCandidate() (string, error) {
	ctx, err := currentDockerContext()
	if err != nil || ctx == nil {
		return "", err
	}
	return dockerHostFromContext(ctx)
}

func dockerHostFromContext(ctx *dockerContextInfo) (string, error) {
	switch {
	case strings.HasPrefix(ctx.Host, "unix://"):
		return ctx.Host, nil
	case strings.HasPrefix(ctx.Host, "tcp://"):
		if ctx.SkipTLSVerify || ctx.HasTLSMaterial {
			return "", fmt.Errorf("current Docker context %q uses a TLS-backed TCP endpoint (%s), but ClawSandbox only auto-detects local unix sockets. Export DOCKER_HOST / DOCKER_TLS_VERIFY / DOCKER_CERT_PATH or switch to a local Docker context",
				ctx.Name, ctx.Host)
		}
		return ctx.Host, nil
	case strings.HasPrefix(ctx.Host, "ssh://"):
		return "", fmt.Errorf("current Docker context %q uses an SSH endpoint (%s), which ClawSandbox does not support. Switch to a local Docker context or expose Docker via a supported socket/TCP endpoint",
			ctx.Name, ctx.Host)
	default:
		return "", fmt.Errorf("current Docker context %q uses an unsupported endpoint (%s)", ctx.Name, ctx.Host)
	}
}

func currentDockerContext() (*dockerContextInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}

	configPath := filepath.Join(home, ".docker", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil
	}

	var cfg struct {
		CurrentContext string `json:"currentContext"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil
	}
	if cfg.CurrentContext == "" || cfg.CurrentContext == "default" {
		return nil, nil
	}

	metaFiles, err := filepath.Glob(filepath.Join(home, ".docker", "contexts", "meta", "*", "meta.json"))
	if err != nil {
		return nil, nil
	}
	for _, metaPath := range metaFiles {
		metaData, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta struct {
			Name      string `json:"Name"`
			Endpoints struct {
				Docker struct {
					Host          string `json:"Host"`
					SkipTLSVerify bool   `json:"SkipTLSVerify"`
				} `json:"docker"`
			} `json:"Endpoints"`
		}
		if err := json.Unmarshal(metaData, &meta); err != nil {
			continue
		}
		if meta.Name == cfg.CurrentContext && meta.Endpoints.Docker.Host != "" {
			contextID := filepath.Base(filepath.Dir(metaPath))
			return &dockerContextInfo{
				Name:           meta.Name,
				Host:           meta.Endpoints.Docker.Host,
				SkipTLSVerify:  meta.Endpoints.Docker.SkipTLSVerify,
				HasTLSMaterial: hasDockerContextTLSMaterial(home, contextID),
			}, nil
		}
	}
	return nil, fmt.Errorf("current Docker context %q is set, but its metadata could not be found", cfg.CurrentContext)
}

func hasDockerContextTLSMaterial(home, contextID string) bool {
	tlsRoot := filepath.Join(home, ".docker", "contexts", "tls", contextID)
	found := false
	_ = filepath.WalkDir(tlsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		found = true
		return filepath.SkipAll
	})
	return found
}
