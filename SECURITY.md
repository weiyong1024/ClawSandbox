# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.0.x   | Yes       |
| < 1.0   | No        |

## Reporting a Vulnerability

If you discover a security vulnerability in ClawFleet, please report it responsibly:

1. **Do NOT open a public issue.**
2. Email **security@clawfleet.io** with a description of the vulnerability, steps to reproduce, and any relevant logs or screenshots.
3. We will acknowledge your report within 48 hours and provide an estimated timeline for a fix.

## Scope

ClawFleet manages Docker containers running OpenClaw instances. Security concerns in scope include:

- Container escape or privilege escalation
- Unauthorized access to the dashboard or API
- Host filesystem exposure through container misconfiguration
- Authentication bypass (Codex OAuth flow)
- Credential leakage in logs or state files

Security issues in upstream OpenClaw itself should be reported to the [OpenClaw project](https://github.com/openclaw/openclaw/security).

## Disclosure

We follow coordinated disclosure. We ask that you give us reasonable time to address the issue before public disclosure.
