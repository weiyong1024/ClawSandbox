# Security Policy

ClawSandbox runs OpenClaw instances with real credentials, browser sessions, and message-channel access. Treat it as security-sensitive infrastructure, not a toy.

## Supported Versions

We currently provide best-effort security support for:

| Version | Supported |
| --- | --- |
| `main` | Yes |
| Latest tagged release | Yes |
| Older releases | No |

## Reporting a Vulnerability

Do not open a public issue for a suspected vulnerability.

Please report it privately through one of these channels:

1. GitHub private vulnerability reporting / security advisory for this repository, if available.
2. Direct maintainer contact through GitHub.

Include:

- affected version or commit
- reproduction steps
- impact assessment
- any proposed mitigation

We will acknowledge the report, investigate, and coordinate disclosure once a fix or mitigation is available.

## Security Model

ClawSandbox is designed for a single operator managing multiple isolated OpenClaw instances on one machine.

The intended baseline is:

- one user-controlled host
- Docker-based instance isolation
- host ports bound to loopback by default
- separate data directories per claw
- separate credentials, accounts, or runtime state when needed

## Operator Responsibilities

ClawSandbox reduces blast radius between claws. It does not remove the need for basic host security.

You are still responsible for:

- protecting the host machine and Docker daemon
- keeping ClawSandbox, OpenClaw, Docker, and the OS up to date
- using separate API keys / bot tokens when isolation matters
- not exposing Dashboard / noVNC / Gateway to the public internet without auth and network controls
- reviewing any skills, prompts, plugins, or scripts you install inside a claw

## Hardening Recommendations

- Keep services on `127.0.0.1` unless you explicitly need remote access.
- Prefer one claw per trust boundary: prod, staging, experiments, rescue, separate channel identities.
- Do not reuse the same browser session or credentials across claws when separation matters.
- Use a dedicated machine account or dedicated phone number for high-trust automations.
- Assume anything with shell, browser, or filesystem access can become high impact if misconfigured.

## Out of Scope

The following are generally not treated as ClawSandbox vulnerabilities by themselves:

- insecure deployment choices by the operator
- exposing services publicly without authentication
- compromise of third-party model providers, messaging platforms, or upstream OpenClaw components
- prompt / skill / plugin logic intentionally granted broad local privileges by the operator
