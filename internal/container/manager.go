package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	cfg "github.com/clawfleet/clawfleet/internal/config"
)

type CreateParams struct {
	Name        string
	ImageRef    string
	NoVNCPort   int
	GatewayPort int
	DataDir     string
	MemoryBytes int64
	NanoCPUs    int64
	RuntimeType string
}

func Create(cli *docker.Client, p CreateParams) (string, error) {
	var (
		portBindings map[docker.Port][]docker.PortBinding
		exposedPorts map[docker.Port]struct{}
		binds        []string
		env          []string
	)

	if p.RuntimeType == "hermes" {
		portBindings = map[docker.Port][]docker.PortBinding{
			"9119/tcp": {{HostIP: "127.0.0.1", HostPort: strconv.Itoa(p.NoVNCPort)}},
			"3000/tcp": {{HostIP: "127.0.0.1", HostPort: strconv.Itoa(p.GatewayPort)}},
		}
		exposedPorts = map[docker.Port]struct{}{
			"9119/tcp": {},
			"3000/tcp": {},
		}
		binds = []string{fmt.Sprintf("%s:/opt/data", p.DataDir)}
		env = []string{
			fmt.Sprintf("HERMES_UID=%d", os.Getuid()),
			fmt.Sprintf("HERMES_GID=%d", os.Getgid()),
		}
	} else {
		portBindings = map[docker.Port][]docker.PortBinding{
			"6901/tcp":  {{HostIP: "127.0.0.1", HostPort: strconv.Itoa(p.NoVNCPort)}},
			"18790/tcp": {{HostIP: "127.0.0.1", HostPort: strconv.Itoa(p.GatewayPort)}},
		}
		exposedPorts = map[docker.Port]struct{}{
			"6901/tcp":  {},
			"18790/tcp": {},
		}
		binds = []string{fmt.Sprintf("%s:/home/node/.openclaw", p.DataDir)}
		env = []string{
			"PLAYWRIGHT_BROWSERS_PATH=/ms-playwright",
		}
	}

	// Hermes: run both dashboard (config UI on :9119) and gateway (messaging on :3000).
	// We override the entrypoint to run setup + multi-process launch in one script.
	// The official entrypoint does: UID/GID remap → gosu drop → dir/config bootstrap
	// → skill sync → exec hermes. We replicate the essential setup steps here.
	var cmd []string
	var entrypoint []string
	if p.RuntimeType == "hermes" {
		entrypoint = []string{"/bin/bash", "-c"}
		cmd = []string{
			`set -e; export HERMES_HOME="${HERMES_HOME:-/opt/data}"; ` +
				// Activate Python venv
				`source /opt/hermes/.venv/bin/activate; ` +
				// Bootstrap dirs and config (same as official entrypoint.sh)
				`mkdir -p "$HERMES_HOME"/{cron,sessions,logs,hooks,memories,skills,skins,plans,workspace,home}; ` +
				`[ -f "$HERMES_HOME/.env" ] || cp /opt/hermes/.env.example "$HERMES_HOME/.env"; ` +
				`[ -f "$HERMES_HOME/config.yaml" ] || cp /opt/hermes/cli-config.yaml.example "$HERMES_HOME/config.yaml"; ` +
				`[ -f "$HERMES_HOME/SOUL.md" ] || cp /opt/hermes/docker/SOUL.md "$HERMES_HOME/SOUL.md"; ` +
				// Sync bundled skills
				`python3 /opt/hermes/tools/skills_sync.py 2>/dev/null || true; ` +
				// Launch dashboard in background, gateway as foreground process
				`hermes dashboard --host 0.0.0.0 --port 9119 --no-open --insecure & ` +
				`exec hermes gateway run`,
		}
	}

	container, err := cli.CreateContainer(docker.CreateContainerOptions{
		Name: p.Name,
		Config: &docker.Config{
			Image:        p.ImageRef,
			Entrypoint:   entrypoint,
			Cmd:          cmd,
			ExposedPorts: exposedPorts,
			Labels:       map[string]string{cfg.LabelManaged: "true"},
			Env:          env,
		},
		HostConfig: &docker.HostConfig{
			Binds:        binds,
			PortBindings: portBindings,
			NetworkMode:  cfg.NetworkName,
			Memory:       p.MemoryBytes,
			NanoCPUs:     p.NanoCPUs,
		},
	})
	if err != nil {
		return "", fmt.Errorf("creating container %s: %w", p.Name, err)
	}
	return container.ID, nil
}

func Start(cli *docker.Client, containerID string) error {
	return cli.StartContainer(containerID, nil)
}

func Stop(cli *docker.Client, containerID string) error {
	return cli.StopContainer(containerID, 10)
}

func Remove(cli *docker.Client, containerID string) error {
	return cli.RemoveContainer(docker.RemoveContainerOptions{
		ID:    containerID,
		Force: true,
	})
}

// IsNotFound returns true if the error indicates the container does not exist.
func IsNotFound(err error) bool {
	_, ok := err.(*docker.NoSuchContainer)
	return ok
}

// Status returns the container's status string and its StartedAt time (zero if not running).
func Status(cli *docker.Client, containerID string) (string, time.Time, error) {
	c, err := cli.InspectContainerWithOptions(docker.InspectContainerOptions{ID: containerID})
	if err != nil {
		return "unknown", time.Time{}, fmt.Errorf("inspecting container %s: %w", containerID, err)
	}
	switch c.State.Status {
	case "running":
		return "running", c.State.StartedAt, nil
	case "exited", "dead":
		return "stopped", time.Time{}, nil
	default:
		return c.State.Status, time.Time{}, nil
	}
}

func Logs(cli *docker.Client, containerID string, follow bool, out io.Writer) error {
	return cli.Logs(docker.LogsOptions{
		Container:    containerID,
		Stdout:       true,
		Stderr:       true,
		Follow:       follow,
		Tail:         "100",
		OutputStream: out,
		ErrorStream:  out,
	})
}

// LogsFollow streams logs with follow=true, cancellable via context.
func LogsFollow(cli *docker.Client, containerID string, ctx context.Context, out io.Writer) error {
	return cli.Logs(docker.LogsOptions{
		Context:      ctx,
		Container:    containerID,
		Stdout:       true,
		Stderr:       true,
		Follow:       true,
		Tail:         "100",
		OutputStream: out,
		ErrorStream:  out,
	})
}

// ParseMemoryBytes converts a human-readable string like "4g", "512m" to bytes.
func ParseMemoryBytes(s string) (int64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	var mul int64 = 1
	switch {
	case strings.HasSuffix(s, "g"):
		mul = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "g")
	case strings.HasSuffix(s, "m"):
		mul = 1024 * 1024
		s = strings.TrimSuffix(s, "m")
	case strings.HasSuffix(s, "k"):
		mul = 1024
		s = strings.TrimSuffix(s, "k")
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory value: %s", s)
	}
	return n * mul, nil
}
