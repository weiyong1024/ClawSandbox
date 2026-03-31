package web

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/clawfleet/clawfleet/internal/state"
)

// OpenAI Codex OAuth constants — matches OpenClaw's registered OAuth application.
const (
	codexClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	codexTokenURL     = "https://auth.openai.com/oauth/token"
	codexRedirectURI  = "http://localhost:1455/auth/callback"
	codexScope        = "openid profile email offline_access"
	codexCallbackPort = 1455
	codexFlowTTL      = 5 * time.Minute
)

// codexStateSep separates the random nonce from the Dashboard origin in the
// OAuth state parameter. The :1455 relay page parses this to know which
// Dashboard API to forward the authorization code to.
const codexStateSep = "."

// codexJWTClaimPath is the JWT claim key that holds auth metadata including the account ID.
const codexJWTClaimPath = "https://api.openai.com/auth"

type codexOAuthResult struct {
	Access    string
	Refresh   string
	Expires   int64
	AccountID string
	Error     string
}

type codexPendingFlow struct {
	verifier  string
	model     string
	name      string
	result    chan *codexOAuthResult
	createdAt time.Time
}

// codexFlowManager manages pending OAuth flows and the :1455 relay server.
// The relay server is stateless — it serves an HTML page that reads code+state
// from the URL and forwards them via fetch() to the Dashboard API encoded in
// the state parameter. This allows a single :1455 listener to serve callbacks
// for multiple Dashboard instances (local + SSH-tunneled remote).
type codexFlowManager struct {
	mu       sync.Mutex
	flows    map[string]*codexPendingFlow // keyed by nonce (first part of state)
	server   *http.Server
	listener net.Listener
}

func newCodexFlowManager() *codexFlowManager {
	return &codexFlowManager{
		flows: make(map[string]*codexPendingFlow),
	}
}

// generatePKCE creates a PKCE code verifier and its S256 challenge.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate random: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// generateNonce creates a random hex nonce for the OAuth state.
func generateNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// buildState encodes the nonce and dashboard origin into the OAuth state param.
// Format: "<nonce>.<origin>" e.g. "a1b2c3.http://localhost:8080"
func buildState(nonce, dashboardOrigin string) string {
	return nonce + codexStateSep + dashboardOrigin
}

// parseState splits the OAuth state into nonce and dashboard origin.
func parseState(state string) (nonce, origin string, ok bool) {
	idx := strings.Index(state, codexStateSep)
	if idx < 1 || idx >= len(state)-1 {
		return "", "", false
	}
	return state[:idx], state[idx+1:], true
}

// buildAuthURL constructs the OpenAI OAuth authorization URL.
func buildAuthURL(stateStr, challenge string) string {
	u, _ := url.Parse(codexAuthorizeURL)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", codexClientID)
	q.Set("redirect_uri", codexRedirectURI)
	q.Set("scope", codexScope)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", stateStr)
	q.Set("id_token_add_organizations", "true")
	q.Set("codex_cli_simplified_flow", "true")
	q.Set("originator", "pi")
	u.RawQuery = q.Encode()
	return u.String()
}

// startCallbackRelay tries to start the :1455 stateless relay server on
// Dashboard startup. The relay is kept running for the lifetime of the
// Dashboard — it serves a single static HTML page with zero ongoing cost.
// If the port is already occupied (by another Dashboard or an SSH tunnel),
// that is fine — the existing listener serves the same relay page or tunnels
// to a remote Dashboard that does.
func (m *codexFlowManager) startCallbackRelay() {
	m.mu.Lock()
	defer m.mu.Unlock()

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", codexCallbackPort))
	if err != nil {
		log.Printf("codex: port %d in use (OK — another listener handles callbacks)", codexCallbackPort)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/callback", handleRelayCallback)
	srv := &http.Server{Handler: mux}

	m.listener = ln
	m.server = srv

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("codex callback relay: %v", err)
		}
	}()
	log.Printf("codex: OAuth callback relay listening on :%d", codexCallbackPort)
}

// handleRelayCallback is the :1455 handler. It serves a static HTML page that
// reads code and state from the URL, parses the Dashboard origin from state,
// and POSTs the code to that Dashboard's /api/v1/oauth/codex/callback endpoint.
// This is a package-level function — it accesses no server state.
func handleRelayCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, codexRelayHTML)
}

// codexRelayHTML is the static HTML page served by the :1455 relay.
// It extracts code + state from the URL, parses the Dashboard origin from
// the state, and forwards the code via fetch() to the correct Dashboard API.
const codexRelayHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Authenticating...</title>
<style>
body{font-family:system-ui,sans-serif;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:#0f172a;color:#e2e8f0}
.card{text-align:center;padding:2rem;border-radius:12px;background:#1e293b;max-width:420px}
h1{margin:0 0 1rem}
.ok{color:#22c55e} .err{color:#ef4444} .wait{color:#facc15}
</style></head>
<body><div class="card">
<h1 class="wait" id="title">Authenticating...</h1>
<p id="msg">Completing login, please wait...</p>
</div>
<script>
(async()=>{
  const title=document.getElementById('title'), msg=document.getElementById('msg');
  const params=new URLSearchParams(location.search);
  const code=params.get('code'), state=params.get('state');
  if(!code||!state){
    title.textContent='Authentication Failed';title.className='err';
    msg.textContent='Missing code or state parameter.';return;
  }
  // State format: "<nonce>.<dashboard_origin>"
  const dot=state.indexOf('.');
  if(dot<1){
    title.textContent='Authentication Failed';title.className='err';
    msg.textContent='Invalid state format.';return;
  }
  const origin=state.substring(dot+1);
  try{
    const res=await fetch(origin+'/api/v1/oauth/codex/callback',{
      method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({code,state})
    });
    const json=await res.json();
    if(res.ok&&json.data?.status==='ok'){
      title.textContent='Authentication Successful';title.className='ok';
      msg.textContent='You can close this window and return to the Dashboard.';
    }else{
      title.textContent='Authentication Failed';title.className='err';
      msg.textContent=json.error?.message||json.data?.error||'Token exchange failed.';
    }
  }catch(e){
    title.textContent='Authentication Failed';title.className='err';
    msg.textContent='Could not reach Dashboard at '+origin+'. '+e.message;
  }
})();
</script></body></html>`

// exchangeCodexToken exchanges an authorization code for tokens.
func exchangeCodexToken(code, verifier string) (*codexOAuthResult, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {codexClientID},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {codexRedirectURI},
	}

	resp, err := httpClient.Post(codexTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		return nil, fmt.Errorf("token response missing required fields")
	}

	accountID := decodeJWTAccountID(tok.AccessToken)
	if accountID == "" {
		return nil, fmt.Errorf("failed to extract account ID from access token")
	}

	return &codexOAuthResult{
		Access:    tok.AccessToken,
		Refresh:   tok.RefreshToken,
		Expires:   time.Now().UnixMilli() + tok.ExpiresIn*1000,
		AccountID: accountID,
	}, nil
}

// decodeJWTAccountID extracts the ChatGPT account ID from a JWT access token.
// Returns empty string on failure (non-fatal).
func decodeJWTAccountID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	auth, ok := claims[codexJWTClaimPath].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := auth["chatgpt_account_id"].(string)
	return id
}

// callbackHTML returns a minimal HTML page for direct server-rendered results.
func callbackHTML(title, message string, isError bool) string {
	color := "#22c55e"
	if isError {
		color = "#ef4444"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:system-ui,sans-serif;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0;background:#0f172a;color:#e2e8f0}
.card{text-align:center;padding:2rem;border-radius:12px;background:#1e293b;max-width:400px}
h1{color:%s;margin:0 0 1rem}</style></head>
<body><div class="card"><h1>%s</h1><p>%s</p></div></body></html>`, title, color, title, message)
}

// --- Server handler methods ---

type codexStartRequest struct {
	Model string `json:"model"`
	Name  string `json:"name"`
}

// handleCodexOAuthStart initiates the Codex OAuth flow.
// POST /api/v1/oauth/codex/start
func (s *Server) handleCodexOAuthStart(w http.ResponseWriter, r *http.Request) {
	var req codexStartRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	verifier, challenge, err := generatePKCE()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	nonce, err := generateNonce()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	flow := &codexPendingFlow{
		verifier:  verifier,
		model:     req.Model,
		name:      req.Name,
		result:    make(chan *codexOAuthResult, 1),
		createdAt: time.Now(),
	}

	s.codexFlows.mu.Lock()
	s.codexFlows.flows[nonce] = flow
	s.codexFlows.mu.Unlock()

	// Derive the Dashboard origin from the incoming request so the :1455 relay
	// knows where to forward the callback. Use the Origin header if present
	// (browser sends it), otherwise reconstruct from Host.
	dashboardOrigin := r.Header.Get("Origin")
	if dashboardOrigin == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		dashboardOrigin = scheme + "://" + r.Host
	}

	stateStr := buildState(nonce, dashboardOrigin)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"auth_url": buildAuthURL(stateStr, challenge),
			"state":    nonce, // Frontend uses the nonce for polling.
		},
	})
}

// codexCallbackRequest is the JSON body POSTed by the :1455 relay page.
type codexCallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"` // Full state: "<nonce>.<origin>"
}

// handleCodexOAuthCallback receives the authorization code from the :1455 relay
// page, exchanges it for tokens, and stores the result for the poll endpoint.
// POST /api/v1/oauth/codex/callback
func (s *Server) handleCodexOAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Allow cross-origin POST from the :1455 relay page.
	w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://localhost:%d", codexCallbackPort))
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var req codexCallbackRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	nonce, _, ok := parseState(req.State)
	if !ok || req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid state or missing code")
		return
	}

	s.codexFlows.mu.Lock()
	flow, found := s.codexFlows.flows[nonce]
	s.codexFlows.mu.Unlock()

	if !found {
		writeError(w, http.StatusNotFound, "unknown or expired OAuth flow")
		return
	}

	// Exchange code for tokens.
	result, err := exchangeCodexToken(req.Code, flow.verifier)
	if err != nil {
		flow.result <- &codexOAuthResult{Error: err.Error()}
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"status": "error", "error": err.Error()},
		})
		return
	}

	flow.result <- result
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"status": "ok"},
	})
}

// handleCodexOAuthPoll checks the status of a pending Codex OAuth flow.
// On completion, creates the model asset and returns it.
// GET /api/v1/oauth/codex/poll?state=XXX
func (s *Server) handleCodexOAuthPoll(w http.ResponseWriter, r *http.Request) {
	nonce := r.URL.Query().Get("state")
	if nonce == "" {
		writeError(w, http.StatusBadRequest, "state parameter required")
		return
	}

	s.codexFlows.mu.Lock()
	flow, ok := s.codexFlows.flows[nonce]
	s.codexFlows.mu.Unlock()

	if !ok {
		writeError(w, http.StatusNotFound, "unknown or expired OAuth flow")
		return
	}

	// Check if flow has expired.
	if time.Since(flow.createdAt) > codexFlowTTL {
		s.codexFlows.mu.Lock()
		delete(s.codexFlows.flows, nonce)
		s.codexFlows.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"status": "failed", "error": "OAuth flow timed out"},
		})
		return
	}

	// Non-blocking check for result.
	select {
	case result := <-flow.result:
		// Clean up flow.
		s.codexFlows.mu.Lock()
		delete(s.codexFlows.flows, nonce)
		s.codexFlows.mu.Unlock()

		if result.Error != "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"data": map[string]any{"status": "failed", "error": result.Error},
			})
			return
		}

		// Create the model asset.
		store, err := s.loadAssets()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		name := flow.name
		if name == "" {
			name = fmt.Sprintf("ChatGPT (Codex) %s", flow.model)
		}

		asset := &state.ModelAsset{
			ID:             generateID(),
			Name:           name,
			Provider:       "openai-codex",
			APIKey:         result.Access,
			Model:          flow.model,
			Validated:      true,
			OAuthRefresh:   result.Refresh,
			OAuthExpires:   result.Expires,
			OAuthAccountID: result.AccountID,
		}

		store.AddModel(asset)
		if err := store.SaveAssets(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{
				"status":     "completed",
				"account_id": result.AccountID,
				"asset":      asset,
			},
		})

	default:
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"status": "pending"},
		})
	}
}
