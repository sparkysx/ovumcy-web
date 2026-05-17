# Contributing to Ovumcy

Thanks for contributing.

## Development Setup

1. Install Go and Node.js.
2. Install frontend deps:

```bash
npm ci
```

3. Run checks locally:

```bash
go test ./...
npm run lint:js
npm run build
```

4. Start app locally:

```bash
go run ./cmd/ovumcy
```

## Reporting Bugs

Before opening a bug, check existing issues:
- https://github.com/ovumcy/ovumcy-web/issues

When opening a bug report, include:
- environment (OS, browser, Go/Node versions),
- exact steps to reproduce,
- expected vs actual behavior,
- relevant logs/screenshots,
- commit hash or branch if testing unreleased code.

Use the bug report template in `.github/ISSUE_TEMPLATE/bug_report.yml`.

Security issues should not be reported publicly. Use [SECURITY.md](SECURITY.md).

## Pull Request Rules

- Keep changes scoped and atomic.
- Add/adjust tests for behavioral changes.
- `internal/i18n/locales/en.json` is the canonical source for UI strings. When you add or rename strings, mirror the change in `ru.json`, `es.json`, `fr.json`, and `de.json` (the five locales advertised as supported in the README). If you cannot provide a native translation for a non-`en` locale, copy the English string verbatim and leave a `TODO(<locale>)` next to it so the gap is visible in review and search.
- Do not introduce legacy compatibility paths unless explicitly required.

## API Stability Contract

`internal/api/routes.go` is the source of truth for HTTP endpoints. There is no separate OpenAPI specification.

The `/api/*` surface is the HTMX backend for the bundled web UI. Endpoints content-negotiate and emit JSON when the client sends `Accept: application/json` or sets `HX-Request: true` with a JSON sub-format, but the JSON shape is **not** treated as a stable third-party API:

- Field additions are non-breaking and may ship in any release.
- Field renames, removals, status code changes, route moves, and error key changes may ship in any minor release without a deprecation cycle.
- The export payload (`POST /api/export/{json,csv,summary}`) is the one exception: it follows the stability contract documented in [docs/export.md](../docs/export.md).

If you are scripting against `/api/*` from outside the bundled UI, pin to a specific image tag and re-validate on every upgrade. If you need a long-term stable API, file an issue describing the use case so we can scope a versioned surface explicitly.

## Commit Style

Use imperative commit messages, e.g.:

- `Fix calendar ovulation tag precedence`
- `Pin staticcheck version in CI`
