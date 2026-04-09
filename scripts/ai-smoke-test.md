# ClawFleet Smoke Test

> Executed by AI via `/smoke-test` skill. All tests are CLI-based.
> CLI success = UI success (Dashboard is a thin REST API client).

## Prerequisites

- Docker running
- ClawFleet binary built (`make build`)
- Dashboard NOT running (test will start its own)
- At least one validated OpenAI model asset with gpt-5-mini (for chat test)

## Execution Rules

1. Run tests in order — later tests depend on earlier ones
2. Each test: print `[PASS] description` or `[FAIL] description + reason`
3. On first FAIL in a critical test (marked with ★), stop and report
4. Track pass/fail counts, print summary at the end
5. Clean up ALL test instances after completion (success or failure)
6. After automated tests, print the Human Verification Checklist

## Test Instance Naming

Use `smoke-1` as the test instance name. If it already exists, destroy it first.

---

## Phase 1: Image (★ critical — blocks everything)

### T1. Image exists or auto-pulls
```bash
# The binary should be able to create an instance (auto-pull if image missing)
# Just verify the image ref is resolvable
curl -sf http://localhost:$DASHBOARD_PORT/api/v1/image/status
```
**Pass:** Response contains `"built": true` or image can be pulled.

### T2. OpenClaw version inside container
```bash
docker exec $INSTANCE openclaw --version
```
**Pass:** Output contains the RecommendedOpenClawVersion from `internal/version/version.go`.

---

## Phase 2: Instance Lifecycle (★ critical)

### T3. Create instance
```bash
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/instances \
  -H 'Content-Type: application/json' -d '{"count":1}'
```
**Pass:** Response contains instance name with status "running".

### T4. List instances
```bash
curl -sf http://localhost:$DASHBOARD_PORT/api/v1/instances
```
**Pass:** Response contains the created instance.

### T5. Stop instance
```bash
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/instances/$INSTANCE/stop
```
**Pass:** 200 OK, instance status changes to "stopped".

### T6. Start instance
```bash
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/instances/$INSTANCE/start
```
**Pass:** 200 OK, instance status changes to "running".

---

## Phase 3: Configuration (★ critical — most bugs surface here)

### T7. Configure with OpenAI gpt-5-mini
Find a validated OpenAI model asset with gpt-5-mini from the asset list.
```bash
# List model assets, find one with provider=openai and model containing "gpt-5-mini"
curl -sf http://localhost:$DASHBOARD_PORT/api/v1/assets/models

# Configure
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/instances/$INSTANCE/configure \
  -H 'Content-Type: application/json' \
  -d '{"model_asset_id":"$MODEL_ASSET_ID"}'
```
**Pass:** Response contains `"status": "configured"`.

### T8. Gateway health check
```bash
docker exec $INSTANCE curl -sf http://127.0.0.1:18789/health
```
**Pass:** Response is `{"ok":true,"status":"live"}`.

### T9. Configuration status
```bash
curl -sf http://localhost:$DASHBOARD_PORT/api/v1/instances/$INSTANCE/configure/status
```
**Pass:** `configured=true`, provider and model match what was configured.

### T10. Chat compatibility (★ critical — catches upstream API bugs)
Wait 10 seconds for Gateway to process heartbeat messages, then check session logs.
```bash
sleep 10
docker exec $INSTANCE bash -c "cat /home/node/.openclaw/agents/main/sessions/*.jsonl 2>/dev/null" \
  | grep -i '"stopReason":"error"'
```
**Pass:** No error entries in session logs. If errors found, extract `errorMessage` and report.
**Known failure pattern:** `reasoning_effort: 'none' is not supported` = upstream OpenClaw bug, not our fault.

---

## Phase 4: Asset Management

### T11. Model asset test endpoint
```bash
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/assets/models/test \
  -H 'Content-Type: application/json' \
  -d '{"provider":"openai","api_key":"$API_KEY","model":"gpt-5-mini"}'
```
**Pass:** Response contains `"valid": true`.
**Note:** Requires a real API key. If no key available, skip with `[SKIP]`.

### T12. Character asset CRUD
```bash
# Create
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/assets/characters \
  -H 'Content-Type: application/json' \
  -d '{"name":"SmokeTestBot","bio":"A test character"}'

# Verify exists in list
curl -sf http://localhost:$DASHBOARD_PORT/api/v1/assets/characters

# Delete
curl -sf -X DELETE http://localhost:$DASHBOARD_PORT/api/v1/assets/characters/$CHAR_ID
```
**Pass:** Create returns 201, character appears in list, delete returns 200.

---

## Phase 5: Roster

### T13. SOUL.md roster injection
Create a temporary character asset, re-configure the test instance with it, then check SOUL.md.
```bash
# Create temp character
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/assets/characters \
  -H 'Content-Type: application/json' \
  -d '{"name":"SmokeTestBot","bio":"A smoke test character"}'
# Extract character ID from response

# Re-configure instance with model + character
curl -sf -X POST http://localhost:$DASHBOARD_PORT/api/v1/instances/$INSTANCE/configure \
  -H 'Content-Type: application/json' \
  -d '{"model_asset_id":"$MODEL_ASSET_ID","character_asset_id":"$CHAR_ID"}'

# Check SOUL.md
docker exec $INSTANCE cat /home/node/.openclaw/workspace/SOUL.md
```
**Pass criteria (check in order):**
1. SOUL.md contains `# SmokeTestBot` (character was injected)
2. If other running instances with characters exist → SOUL.md also contains `## Your Team`
3. If no other instances have characters → no `## Your Team` section (also pass)
4. Instance without a character asset → SOUL.md is OpenClaw's default content (also pass, but T13 should always configure with a character)

**Cleanup:** Delete the temp character asset after the test.

---

## Phase 6: Control Panel

### T14. Gateway port directly accessible
```bash
# Get gateway port from instance list
curl -sf http://localhost:$GATEWAY_PORT/
```
**Pass:** HTTP 200, response contains `<title>OpenClaw Control</title>`.

### T15. Gateway assets load
```bash
# Extract a JS asset path from the HTML, verify it loads
JS_PATH=$(curl -sf http://localhost:$GATEWAY_PORT/ | grep -o 'src="./assets/[^"]*' | head -1 | sed 's/src=".//')
curl -sf -o /dev/null -w "%{http_code}" http://localhost:$GATEWAY_PORT$JS_PATH
```
**Pass:** HTTP 200.

### T16. WebSocket upgrade
```bash
curl -s -o /dev/null -w "%{http_code}" --max-time 3 \
  -H "Upgrade: websocket" -H "Connection: Upgrade" \
  -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" -H "Sec-WebSocket-Version: 13" \
  http://localhost:$GATEWAY_PORT/
```
**Pass:** HTTP 101.

---

## Phase 7: Cleanup

### T17. Destroy test instance
```bash
curl -sf -X DELETE http://localhost:$DASHBOARD_PORT/api/v1/instances/$INSTANCE
```
**Pass:** Instance destroyed.

### T18. Verify cleanup
```bash
curl -sf http://localhost:$DASHBOARD_PORT/api/v1/instances
```
**Pass:** Test instance no longer in list.

---

## Summary

Print results:
```
═══════════════════════════════════════
  ClawFleet Smoke Test Results
═══════════════════════════════════════
  Passed: XX / 18
  Failed: XX
  Skipped: XX
═══════════════════════════════════════
```

If any FAIL, list them with reasons.

---

## Human Verification Checklist

Print this ONLY if all automated tests passed:

```
═══════════════════════════════════════
  All automated tests passed! ✓

  Please verify the following manually:

  1. Open Dashboard: http://localhost:8080
     → Instance cards display correctly
     → Model/Channel/Character assets show in sidebar

  2. Open Control Panel for any configured instance:
     http://localhost:{gateway_port}/
     → Click Connect
     → Send "hello" → bot responds

  3. (If Codex OAuth available) Add Model Config:
     → Select ChatGPT (Codex)
     → Click "Login with ChatGPT"
     → Complete OAuth flow → asset created

  Report results: all pass / which failed
═══════════════════════════════════════
```
