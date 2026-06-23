# Contributing to ZeroDrop Terminal

First off, thank you for considering contributing! ZeroDrop is a security-focused
application and every contribution helps make credential delivery safer.

## Code of Conduct

This project and everyone participating in it is governed by the
[Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to
uphold this code.

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/your-org/zerodrop/issues)
2. If not, open a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, browser version)
   - Logs (if applicable)

### Suggesting Features

1. Open an issue with the label `enhancement`
2. Describe the problem you're trying to solve
3. Explain your proposed solution
4. Note any security implications

### Security Vulnerabilities

**Do NOT open public issues for security vulnerabilities.**
See [SECURITY.md](SECURITY.md) for our disclosure policy.

## Development Setup

### Prerequisites

- Go 1.26+
- Node.js 20+
- Docker 20.10+ (for containerized deployment)
- Make

### Local Development

```bash
# Clone the repository
git clone https://github.com/your-org/zerodrop.git
cd zerodrop

# Run tests
make test

# Run tests with race detection
make test-race

# Build backend
make build-backend

# Run with Mock Printer (no hardware needed)
make dev

# For frontend development
make dev-frontend
```

### Coding Standards

ZeroDrop follows Go and TypeScript coding standards defined in
`.claude/CODING_STANDARDS.md`. Key points:

- **Go**: `gofmt` formatting, `go vet` compliance, error wrapping with `%w`
- **TypeScript**: Strict mode, ESLint + Prettier formatting
- **Security**: All sensitive buffers must be zeroed after use
- **Zero-knowledge**: Server must never possess plaintext or private key

## Pull Request Process

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** following the coding standards

3. **Run all checks** locally before committing:
   ```bash
   make check           # Dependency checks
   make test-race       # Tests with race detection
   make check-security  # Security verification suite
   go vet ./...         # Static analysis
   go fmt ./...         # Formatting
   ```

4. **Write tests** for any new functionality

5. **Update documentation** if you change behavior or add features

6. **Ensure the zero-knowledge guarantee** is preserved:
   - Server must never possess plaintext payload or private key
   - All sensitive memory must be zeroed after use
   - No new database dependencies

7. **Create a pull request** against `main`:
   - Clear title and description
   - Reference any related issues
   - Note any security considerations

## CI/CD

All pull requests and pushes to main trigger GitHub Actions CI:
- Backend: `go vet`, `go test -race`, security checks, build
- Frontend: TypeScript type check, ESLint, production build
- Docker: Build image, health check, API smoke test

## Code Review

All submissions require review. Reviewers will check:
- Correctness and security
- Test coverage
- Documentation
- Coding standards compliance
- Zero-knowledge guarantee preservation

## Branch Naming

- `feature/` — New features
- `fix/` — Bug fixes
- `chore/` — Maintenance, dependencies, tooling
- `docs/` — Documentation changes

## Commit Messages

Use conventional commits:
```
feat: add USB printer auto-detection
fix: resolve race condition in spooler metrics
docs: update README with deployment instructions
test: add regression test for Burn Protocol
refactor: simplify crypto API surface
chore: update Go dependencies
```

## Questions?

Open a [Discussion](https://github.com/your-org/zerodrop/discussions) or
ask in your pull request comments.

Thank you for contributing to making credential delivery safer! 🛡️
