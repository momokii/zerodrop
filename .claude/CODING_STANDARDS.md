# Coding Standards — ZeroDrop Terminal

> **Go-specific conventions** for ZeroDrop Terminal v1.0 security application.

---

## General Principles

- **Clarity over cleverness** — code is read far more often than it is written; optimize for the reader
- **One responsibility per function, file, and module** — if a function does two things, split it
- **Explicit is better than implicit** — no magic, no surprising side effects, no hidden state
- **Fail fast and loudly** — surface errors at the earliest possible point; silent failures are bugs
- **Composition over inheritance** — prefer small, composable functions and modules over deep hierarchies
- **Dry is a guideline, not a religion** — three similar lines are better than a premature abstraction

---

## Go-Specific Naming Conventions

### Files

- Use `snake_case` for all Go files: `crypto.go`, `printer_test.go`, `http_handler.go`
- Test files: `<name>_test.go` in the same package as the code under test
- Package directories: short, lowercase, single-word when possible: `pkg/crypto/`, `pkg/api/`

### Functions & Methods

- **Exported functions**: `PascalCase` — `GenerateKeyPair`, `PrintPayload`, `HandleDrop`
- **Unexported functions**: `camelCase` — `zeroBuffer`, `validateInput`, `parseHeader`
- Descriptive verb-noun pairs: `getUserById`, `validateInput`, `processOrder`
- Avoid generic names: `handle`, `process`, `run` (too vague without context)

### Interfaces

- **Interface names**: `-er` suffix for single-method interfaces: `Printer`, `Spooler`, `Logger`
- Multi-method interfaces: descriptive PascalCase: `HTTPHandler`, `KeyManager`
- **Provider interfaces**: Use `Provider` suffix for interfaces that resolve dependencies at runtime: `PrinterProvider` (returns current active `Printer`)

### Variables

- **Exported constants**: `PascalCase` or `UPPER_SNAKE_CASE` — `MaxRetries`, `BATCH_SIZE`
- **Unexported constants**: `camelCase` — `defaultTimeout`, `maxWorkers`
- **Local variables**: `camelCase` — `publicKey`, `ciphertext`, `printJob`
- **Boolean variables**: prefix with `is`, `has`, `should`, `can`: `isValid`, `hasPermission`, `shouldRetry`
- **Error variables**: `err` — always name error returns `err`

### Structs & Types

- **Exported structs**: `PascalCase` — `Server`, `Config`, `DropRequest`
- **Unexported structs**: `camelCase` with leading `camelCase` — `spoolerWorker`, `httpHandler`
- **Field names**:
  - Exported fields: `PascalCase` — `PublicKey`, `Ciphertext`, `Timestamp`
  - Unexported fields: `camelCase` — `privateKey`, `bufferPool`, `logger`

### Packages

- Short, lowercase, single-word: `crypto`, `api`, `printer`, `spooler`, `config`, `observability`
- Avoid underscores, mixed case, or abbreviations

---

## Go-Specific Error Handling

- **Never silently swallow errors** — every error must be handled explicitly
- **All errors must be logged with sufficient context** — include inputs, state, and operation name
- **User-facing errors must never expose internal details** — return generic messages, log details
- **Use `errors.Is()` and `errors.As()`** for error comparison and type checking
- **Wrap errors with context** using `fmt.Errorf("operation failed: %w", err)` or `errors.Wrap()`
- **Return early on errors** — use guard clauses: `if err != nil { return err }`
- **Distinguish between expected failures and unexpected errors**:
  - Expected: business logic (return error with context)
  - Unexpected: bugs (log and fail)

---

## Go Code Structure

- **Keep functions short** — if it doesn't fit on one screen (~50 lines), it's doing too much
- **Minimize mutation** — prefer returning new values over mutating arguments
- **Guard clauses over deep nesting** — check preconditions early and return, then continue with happy path
- **Group related code together** — colocate functions that operate on the same data
- **Separate concerns by layer:** handler → service → data — never skip layers
- **Use table-driven tests** for multiple test cases
- **Defer cleanup** — use `defer` for cleanup: `defer file.Close()`, `defer zeroBuffer(&key)`

---

## Go-Specific Testing

- **Every new feature must include at least one test**
- **Every bug fix must include a regression test**
- **Test command:** `go test ./... -v -race -cover`
- **Test coverage target:** 80%+ for security-critical code (`pkg/crypto/`, `pkg/api/`)
- **Benchmark tests:** use `_bench_test.go` suffix, run with `go test -bench=.`
- **Race detection:** always run tests with `-race` flag
- **Test structure:** Arrange → Act → Assert — make expected behavior obvious
- **Table-driven tests** for multiple test cases:
  ```go
  tests := []struct {
      name string
      input string
      want string
      wantErr bool
  }{
      {"valid input", "test", "result", false},
      {"invalid input", "", "", true},
  }
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { /* test */ })
  }
  ```

---

## Go Documentation

- **Every exported function must have a godoc comment** — what it does, what it returns, what errors
- **Package comments** at the top of each package file: `// Package crypto provides ECC key generation...`
- **Godoc format:**
  ```go
  // GenerateKeyPair creates a new X25519 key pair.
  // Returns the public key in PEM format and logs the private key as a QR code.
  // The private key is saved to disk (0600) and only loaded to RAM during first-run QR display.
  func GenerateKeyPair() ([]byte, error)
  ```
- **Every non-obvious decision in code must have an inline comment explaining *why*** — not *what* (the code says what)
- **Keep comments current** — an outdated comment is worse than no comment
- **No commented-out code** — delete it; version control remembers everything

---

## Go Tooling & Linting

- **Format code:** `go fmt ./...` (or `gofmt -w .`)
- **Lint:** `golangci-lint run` (recommended linter)
- **Vet:** `go vet ./...` (catches common mistakes)
- **Imports:** group imports in three sections (stdlib, third-party, internal), separate with blank line
- **Mod tidy:** `go mod tidy` after adding/removing dependencies

### Recommended golangci-lint configuration

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - gosec  # security-focused linter (critical for ZeroDrop)
```

---

## Go-Specific Security Patterns

- **Never log sensitive data** — no keys, ciphertext, plaintext in logs
- **Use `crypto/rand`** — never `math/rand` for security-critical randomness
- **Constant-time comparison** — use `crypto/subtle.ConstantTimeCompare()` for secrets (admin token, auth tokens)
- **Zero buffers after use** — implement Burn Protocol with `runtime.KeepAlive()`
- **Validate all input at boundaries** — HTTP handlers should validate before passing to services
- **Use `context.Context`** for all operations that may timeout or cancel
- **Avoid reflection** in security-critical code — it can bypass access controls
- **Thread-safe metrics** — use `sync/atomic` for counters, `sync.Mutex` for complex state snapshots
- **PrinterProvider pattern** — spooler resolves printer per-job via `PrinterProvider` interface instead of holding a direct reference, enabling runtime printer switching

---

## Git Conventions

- **Commit messages:** Use conventional commit format:
  - `feat: add USB printer implementation`
  - `fix: resolve race condition in spooler`
  - `docs: update README with deployment instructions`
  - `test: add regression test for Burn Protocol`
  - `refactor: simplify crypto API surface`
- **Branch naming:** `feature/`, `fix/`, `chore/` prefixes with descriptive names
- **No direct commits to main/master** — always use branches for non-trivial changes
