# AGENTS.md

## Project build and release workflow

The `grapes` CLI entry point lives at `./cmd/grapes`.

### Local feature testing during development

Prefer `go run` so day-to-day testing does not leave build artifacts in the repository:

```bash
go run ./cmd/grapes <source.grapes>
go run ./cmd/grapes <source.grapes> -t zsh
```

### Local executable builds

If you need a saved local binary, build it into `./bin` instead of the repository root:

```bash
go build -o ./bin/grapes ./cmd/grapes
```

### Release and packaging verification

Use GoReleaser when you want to verify release packaging or produce cross-platform artifacts in `./dist`:

```bash
goreleaser release --snapshot --clean
```

The repository's `.goreleaser.yml` builds `./cmd/grapes` for `linux`, `darwin`, and `windows` on `amd64` and `arm64`, packaging archives plus a SHA-256 checksum file.
