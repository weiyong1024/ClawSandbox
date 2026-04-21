# Hermes Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable ClawFleet to create and manage Hermes containers alongside OpenClaw in a unified Fleet.

**Architecture:** Hermes is lifecycle-only — ClawFleet manages container create/start/stop/destroy and exposes Hermes native ports. OpenClaw code paths are untouched. RuntimeType field on Instance drives branching at create time and UI rendering.

**Tech Stack:** Go, Preact (htm), Docker API (go-dockerclient)

---

### Task 1: Data Model — Instance.RuntimeType

**Files:**
- Modify: `internal/state/store.go:19-28`

- [ ] **Step 1: Add RuntimeType to Instance struct**

```go
type Instance struct {
	Name             string    `json:"name"`
	ContainerID      string    `json:"container_id"`
	Status           string    `json:"status"`
	Ports            Ports     `json:"ports"`
	CreatedAt        time.Time `json:"created_at"`
	ModelAssetID     string    `json:"model_asset_id,omitempty"`
	ChannelAssetID   string    `json:"channel_asset_id,omitempty"`
	CharacterAssetID string    `json:"character_asset_id,omitempty"`
	RuntimeType      string    `json:"runtime_type,omitempty"`
}
```

- [ ] **Step 2: Add IsHermes helper**

```go
func (inst *Instance) IsHermes() bool {
	return inst.RuntimeType == "hermes"
}
```

- [ ] **Step 3: Build and run existing tests**

Run: `make build && make test`
Expected: All pass — field is omitempty, backward compatible with existing state.json.

- [ ] **Step 4: Commit**

```bash
git add internal/state/store.go
git commit -m "feat: add RuntimeType field to Instance for multi-runtime support"
```

---

### Task 2: Config — Hermes Image Defaults

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add HermesConfig and constants**

Add after existing constants:
```go
const (
	DefaultHermesImageName = "nousresearch/hermes-agent"
	DefaultHermesImageTag  = "latest"
)
```

Add HermesConfig struct after NamingConfig:
```go
type HermesConfig struct {
	ImageName string `yaml:"image_name"`
	ImageTag  string `yaml:"image_tag"`
}
```

Add Hermes field to Config struct:
```go
type Config struct {
	Image     ImageConfig    `yaml:"image"`
	Hermes    HermesConfig   `yaml:"hermes"`
	Ports     PortsConfig    `yaml:"ports"`
	Resources ResourceConfig `yaml:"resources"`
	Naming    NamingConfig   `yaml:"naming"`
}
```

Add HermesImageRef method:
```go
func (c *Config) HermesImageRef() string {
	return fmt.Sprintf("%s:%s", c.Hermes.ImageName, c.Hermes.ImageTag)
}
```

Update DefaultConfig:
```go
func DefaultConfig() *Config {
	return &Config{
		Image:  ImageConfig{Name: DefaultImageName, Tag: version.ImageTag()},
		Hermes: HermesConfig{ImageName: DefaultHermesImageName, ImageTag: DefaultHermesImageTag},
		Ports:  PortsConfig{NoVNCBase: DefaultNoVNCBase, GatewayBase: DefaultGatewayBase},
		Resources: ResourceConfig{
			MemoryLimit: DefaultMemoryLimit,
			CPULimit:    DefaultCPULimit,
		},
		Naming: NamingConfig{Prefix: DefaultNamingPrefix},
	}
}
```

- [ ] **Step 2: Build**

Run: `make build`
Expected: Compiles clean.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add Hermes image config with defaults"
```

---

### Task 3: Container Manager — Runtime-Aware Creation

**Files:**
- Modify: `internal/container/manager.go`

- [ ] **Step 1: Add RuntimeType to CreateParams**

```go
type CreateParams struct {
	Name        string
	ImageRef    string
	NoVNCPort   int
	GatewayPort int
	DataDir     string
	MemoryBytes int64
	NanoCPUs    int64
	RuntimeType string // "openclaw" or "hermes"
}
```

- [ ] **Step 2: Make Create() runtime-aware**

Replace the hardcoded port bindings and volume binds in `Create()`:

```go
func Create(cli *docker.Client, p CreateParams) (string, error) {
	var portBindings map[docker.Port][]docker.PortBinding
	var exposedPorts map[docker.Port]struct{}
	var binds []string
	var env []string

	if p.RuntimeType == "hermes" {
		// Hermes: Dashboard (9119) + Gateway (3000)
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
		// OpenClaw: noVNC (6901) + Gateway bridge (18790)
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

	container, err := cli.CreateContainer(docker.CreateContainerOptions{
		Name: p.Name,
		Config: &docker.Config{
			Image:        p.ImageRef,
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
```

- [ ] **Step 3: Add `os` import**

Add `"os"` to imports.

- [ ] **Step 4: Build**

Run: `make build`
Expected: Compiles clean.

- [ ] **Step 5: Commit**

```bash
git add internal/container/manager.go
git commit -m "feat: runtime-aware container creation (OpenClaw + Hermes ports/volumes)"
```

---

### Task 4: Create Handler — Runtime Type Passthrough

**Files:**
- Modify: `internal/web/handlers.go`

- [ ] **Step 1: Add RuntimeType to createRequest**

```go
type createRequest struct {
	Count        int    `json:"count"`
	SnapshotName string `json:"snapshot_name,omitempty"`
	RuntimeType  string `json:"runtime_type,omitempty"`
}
```

- [ ] **Step 2: Update handleCreateInstances — image selection and data dir**

In `handleCreateInstances`, after parsing the request, add image selection logic. Replace the current image check block and instance creation loop with runtime-aware logic:

After `cfg := s.config`, add:
```go
	runtimeType := req.RuntimeType
	if runtimeType == "" {
		runtimeType = "openclaw"
	}

	// Select image based on runtime
	var imageRef, imageName, imageTag string
	if runtimeType == "hermes" {
		imageRef = cfg.HermesImageRef()
		imageName = cfg.Hermes.ImageName
		imageTag = cfg.Hermes.ImageTag
	} else {
		imageRef = cfg.ImageRef()
		imageName = cfg.Image.Name
		imageTag = cfg.Image.Tag
	}
```

Replace `container.ImageExists(s.docker, cfg.ImageRef())` with `container.ImageExists(s.docker, imageRef)` and update the pull call to use `imageName, imageTag`.

In the instance creation loop, update the data directory:
```go
	dataSuffix := "openclaw"
	if runtimeType == "hermes" {
		dataSuffix = "hermes"
	}
	instanceDataDir := filepath.Join(dataDir, "data", name, dataSuffix)
```

Disable snapshots for Hermes:
```go
	if req.SnapshotName != "" && runtimeType == "hermes" {
		writeError(w, http.StatusBadRequest, "Soul Archive is not supported for Hermes instances")
		return
	}
```

Pass RuntimeType to CreateParams and to Instance:
```go
	containerID, err := container.Create(s.docker, container.CreateParams{
		// ... existing fields
		RuntimeType: runtimeType,
	})
```

```go
	inst := &state.Instance{
		// ... existing fields
		RuntimeType: runtimeType,
	}
```

- [ ] **Step 3: Build**

Run: `make build`
Expected: Compiles clean.

- [ ] **Step 4: Commit**

```bash
git add internal/web/handlers.go
git commit -m "feat: create handler supports runtime_type parameter"
```

---

### Task 5: Handler Guards — Reject OpenClaw-Only Ops for Hermes

**Files:**
- Modify: `internal/web/handlers_configure.go`
- Modify: `internal/web/handlers_skills.go`
- Modify: `internal/web/handlers.go` (restart-bot, reset)

- [ ] **Step 1: Add guard to handleConfigureInstance**

At the top of `handleConfigureInstance`, after loading the instance:
```go
	if inst.IsHermes() {
		writeError(w, http.StatusBadRequest,
			"Not available for Hermes instances. Use the Hermes Dashboard to configure.")
		return
	}
```

- [ ] **Step 2: Add guard to handleRestartBot**

Same pattern at the top of `handleRestartBot`.

- [ ] **Step 3: Add guard to all skills handlers**

Add the same guard to `handleListInstanceSkills`, `handleInstallSkill`, `handleUninstallSkill`.

- [ ] **Step 4: Add guard to handleResetInstance**

Same pattern.

- [ ] **Step 5: Build and test**

Run: `make build && make test`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add internal/web/handlers_configure.go internal/web/handlers_skills.go internal/web/handlers.go
git commit -m "feat: reject OpenClaw-only operations for Hermes instances"
```

---

### Task 6: Instance List Response — Runtime-Specific Ports

**Files:**
- Modify: `internal/web/handlers.go` (instanceResponse and handleListInstances)

- [ ] **Step 1: Add runtime fields to instanceResponse**

```go
type instanceResponse struct {
	// ... existing fields
	RuntimeType        string `json:"runtime_type,omitempty"`
	HermesDashboardPort int   `json:"hermes_dashboard_port,omitempty"`
	HermesGatewayPort   int   `json:"hermes_gateway_port,omitempty"`
}
```

- [ ] **Step 2: Populate runtime fields in handleListInstances**

When building instanceResponse, add:
```go
	resp.RuntimeType = inst.RuntimeType
	if inst.IsHermes() {
		resp.HermesDashboardPort = inst.Ports.NoVNC   // reuse NoVNC slot for Hermes Dashboard
		resp.HermesGatewayPort = inst.Ports.Gateway     // reuse Gateway slot for Hermes Gateway
	}
```

- [ ] **Step 3: Build**

Run: `make build`

- [ ] **Step 4: Commit**

```bash
git add internal/web/handlers.go
git commit -m "feat: instance list includes runtime type and Hermes port info"
```

---

### Task 7: Frontend — Create Dialog Runtime Dropdown

**Files:**
- Modify: `internal/web/static/js/components/create-dialog.js`
- Modify: `internal/web/static/js/api.js`
- Modify: `internal/web/static/js/i18n.js`

- [ ] **Step 1: Add i18n strings**

English:
```js
'create.runtime': 'Runtime',
'create.runtimeOpenClaw': 'OpenClaw',
'create.runtimeHermes': 'Hermes',
```

Chinese:
```js
'create.runtime': '运行时',
'create.runtimeOpenClaw': 'OpenClaw',
'create.runtimeHermes': 'Hermes',
```

- [ ] **Step 2: Add runtime state and dropdown to create-dialog.js**

Add state: `const [runtime, setRuntime] = useState('openclaw');`

Add dropdown before instance count:
```js
<label class="form-label">
  ${t('create.runtime')}
  <select class="form-input" value=${runtime} onChange=${(e) => setRuntime(e.target.value)}>
    <option value="openclaw">${t('create.runtimeOpenClaw')}</option>
    <option value="hermes">${t('create.runtimeHermes')}</option>
  </select>
</label>
```

Disable snapshot dropdown when runtime is hermes.

- [ ] **Step 3: Pass runtime_type in API call**

In api.js, update `createInstances`:
```js
createInstances: (count, snapshotName, runtimeType) =>
  request('POST', '/instances', {
    count,
    ...(snapshotName && { snapshot_name: snapshotName }),
    ...(runtimeType && runtimeType !== 'openclaw' && { runtime_type: runtimeType }),
  }),
```

Update the call site in create-dialog.js to pass `runtime`.

- [ ] **Step 4: Build**

Run: `make build`

- [ ] **Step 5: Commit**

```bash
git add internal/web/static/js/components/create-dialog.js internal/web/static/js/api.js internal/web/static/js/i18n.js
git commit -m "feat: create dialog with runtime selector (OpenClaw / Hermes)"
```

---

### Task 8: Frontend — Instance Card Runtime Badge + Conditional Buttons

**Files:**
- Modify: `internal/web/static/js/components/instance-card.js`
- Modify: `internal/web/static/js/app.js`
- Modify: `internal/web/static/js/i18n.js`

- [ ] **Step 1: Add i18n strings**

English:
```js
'card.hermesDashboard': '⚕ Dashboard',
'runtime.openclaw': 'OpenClaw',
'runtime.hermes': 'Hermes',
```

Chinese:
```js
'card.hermesDashboard': '⚕ 控制台',
'runtime.openclaw': 'OpenClaw',
'runtime.hermes': 'Hermes',
```

- [ ] **Step 2: Update instance-card.js**

Add runtime badge next to instance name:
```js
${instance.runtime_type === 'hermes'
  ? html`<span class="runtime-badge runtime-hermes">☤ Hermes</span>`
  : html`<span class="runtime-badge runtime-openclaw">🦞</span>`
}
```

Conditionally show buttons:
```js
${!isHermes && html`
  <button class="btn btn-sm btn-desktop" onClick=${onDesktop}>Desktop</button>
  <button class="btn btn-sm btn-desktop" onClick=${onConsole}>Control Panel</button>
  <button class="btn btn-sm" onClick=${onConfigure}>Configure</button>
`}
${isHermes && html`
  <button class="btn btn-sm btn-desktop" onClick=${onHermesDashboard}>⚕ Dashboard</button>
`}
```

Where `isHermes = instance.runtime_type === 'hermes'`.

Show Skills, Save Soul, Restart Bot only for OpenClaw:
```js
${!isHermes && html`
  <button ...>Skills</button>
  <button ...>Save Soul</button>
  <button ...>Restart Bot</button>
`}
```

- [ ] **Step 3: Add onHermesDashboard handler in app.js**

```js
const onHermesDashboard = (name) => {
  const inst = instances.find(i => i.name === name);
  if (inst && inst.hermes_dashboard_port) {
    window.open(`http://localhost:${inst.hermes_dashboard_port}/`, '_blank');
  }
};
```

Pass it to Dashboard and InstanceCard components.

- [ ] **Step 4: Add CSS for runtime badge**

In `style.css`:
```css
.runtime-badge {
  font-size: 0.65rem;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 8px;
  font-weight: 500;
}
.runtime-hermes {
  background: rgba(76, 175, 80, 0.2);
  color: #4caf50;
}
.runtime-openclaw {
  opacity: 0.5;
}
```

- [ ] **Step 5: Build**

Run: `make build`

- [ ] **Step 6: Commit**

```bash
git add internal/web/static/js/components/instance-card.js internal/web/static/js/app.js internal/web/static/js/i18n.js internal/web/static/css/style.css
git commit -m "feat: instance card with runtime badge and conditional buttons"
```

---

### Task 9: Hermes Auto-Pull + Image Status

**Files:**
- Modify: `internal/web/handlers.go` (create handler — already done in Task 4)
- Modify: `internal/web/handlers_image.go`

- [ ] **Step 1: Update handleImageStatus to include Hermes**

Return both OpenClaw and Hermes image status:
```go
func (s *Server) handleImageStatus(w http.ResponseWriter, r *http.Request) {
	openclawExists, _ := container.ImageExists(s.docker, s.config.ImageRef())
	hermesExists, _ := container.ImageExists(s.docker, s.config.HermesImageRef())
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"image":        s.config.ImageRef(),
			"built":        openclawExists,
			"hermes_image": s.config.HermesImageRef(),
			"hermes_built": hermesExists,
		},
	})
}
```

- [ ] **Step 2: Build**

Run: `make build`

- [ ] **Step 3: Commit**

```bash
git add internal/web/handlers_image.go
git commit -m "feat: image status includes Hermes image availability"
```

---

### Task 10: Smoke Test

- [ ] **Step 1: Build and start Dashboard**

```bash
make build
./bin/clawfleet dashboard serve &
```

- [ ] **Step 2: Create OpenClaw instance — verify zero regression**

```bash
curl -sf -X POST http://localhost:8080/api/v1/instances \
  -H 'Content-Type: application/json' -d '{"count":1}'
```

Verify: status=running, has novnc_port and gateway_port, runtime_type empty or "openclaw".

- [ ] **Step 3: Create Hermes instance**

```bash
curl -sf -X POST http://localhost:8080/api/v1/instances \
  -H 'Content-Type: application/json' -d '{"count":1,"runtime_type":"hermes"}'
```

Verify: status=running, has hermes_dashboard_port and hermes_gateway_port.

- [ ] **Step 4: Verify Hermes Dashboard accessible**

```bash
curl -sf -o /dev/null -w "%{http_code}" http://localhost:{hermes_dashboard_port}/
```

Expected: 200

- [ ] **Step 5: Verify OpenClaw-only ops rejected for Hermes**

```bash
curl -sf -X POST http://localhost:8080/api/v1/instances/{hermes-name}/configure \
  -H 'Content-Type: application/json' -d '{"model_asset_id":"any"}'
```

Expected: 400 with "Not available for Hermes instances"

- [ ] **Step 6: Verify mixed Fleet list**

```bash
curl -sf http://localhost:8080/api/v1/instances
```

Expected: Both instances in list, different runtime_type values.

- [ ] **Step 7: Clean up**

```bash
curl -sf -X POST http://localhost:8080/api/v1/instances/batch-destroy \
  -H 'Content-Type: application/json' -d '{"names":["claw-1","claw-2"]}'
```

- [ ] **Step 8: Commit test results**

```bash
git commit --allow-empty -m "test: Hermes integration smoke test passed"
```
