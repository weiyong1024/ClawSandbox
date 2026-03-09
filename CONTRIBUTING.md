# Contributing

ClawSandbox is early-stage infrastructure software. Small, focused contributions are preferred over broad refactors.

## Before You Start

- Open an issue for larger changes, behavior changes, or architecture proposals.
- Keep PRs scoped to one concern when possible.
- Update docs when user-facing behavior changes.

## Local Setup

```bash
git clone https://github.com/weiyong1024/ClawSandbox.git
cd ClawSandbox
make build
./bin/clawsandbox version
```

Useful commands:

```bash
make test
make vet
make build-all
```

If you are working on the runtime image:

```bash
make docker-build
```

## Development Guidelines

- Prefer clear, boring code over clever code.
- Keep config and CLI behavior backward compatible unless there is a strong reason not to.
- Preserve the project's local-first and isolation-first design.
- Avoid adding heavyweight dependencies without justification.
- For Docker- or system-facing changes, include manual verification steps in the PR description.

## Pull Request Checklist

Before opening a PR, make sure you have:

- built the CLI successfully
- run tests and vet locally
- updated relevant docs and examples
- described the user-visible change
- called out follow-up work or known limitations

## Areas That Need Help

- test coverage
- security hardening
- dashboard UX polish
- Docker image optimization
- documentation accuracy and examples

## Reporting Bugs

Please include:

- OS and architecture
- Docker environment
- ClawSandbox version or commit
- reproduction steps
- logs or screenshots if relevant

For security issues, use the process in [SECURITY.md](./SECURITY.md) instead of a public issue.
