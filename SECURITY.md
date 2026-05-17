# Security Policy

## Supported Versions

Security fixes are provided for the `main` branch only.

| Version | Supported |
| --- | --- |
| `main` | :white_check_mark: |
| older commits/tags | :x: |

## Reporting a Vulnerability

Please report security issues privately.

- Email: `contact@ovumcy.com`
- Subject: `SECURITY: <short summary>`
- Include: impact, reproduction steps, affected endpoints/files, and suggested fix if available

We will acknowledge receipt within 72 hours and provide a remediation plan after triage.

Do not open public GitHub issues for unpatched security vulnerabilities.

## Known Information Disclosure

A few information-disclosure signals are accepted residual risk because closing them either breaks core flows or requires infrastructure Ovumcy does not assume self-hosters have (for example SMTP).

### Register enumeration: pickup-cookie follow-up oracle (residual)

`POST /api/auth/register` returns identical status, body, and Set-Cookie shape (a single sealed `ovumcy_register_pickup` cookie of fixed length, no `ovumcy_auth` or `ovumcy_recovery_code`) for both a brand-new email and a duplicate. Response timing is also equalized: the duplicate-email branch runs `equalizeRegistrationTiming` (two bcrypt-cost-10 comparisons against fixed placeholder hashes) to match the work `BuildOwnerUserWithRecovery` performs on the new-email branch (password hash + recovery-code hash), so the single POST response carries no observable difference. The follow-up `GET /register/welcome` then dispatches the pickup cookie either to `/register` (with auth and recovery cookies) for a valid pickup or to `/login` (with a neutral flash) for a decoy or expired pickup. An attacker who holds their own pickup cookie from a probe POST can therefore observe which redirect target their cookie resolves to and infer whether the email was new or already registered. This turns the single-request status / cookie / timing oracle into a two-request probe, with both endpoints under per-IP rate limiting, and the login bcrypt-timing equalization (`equalizeAuthCredentialsTiming`) further bounding any cross-endpoint follow-up.

Closing the residual signal entirely is mathematically impossible without an out-of-band verification channel: any in-app dispatch on the pickup cookie reveals the branch, and any login-after-register variant turns the probe into a follow-up POST `/api/auth/login` whose success or failure carries the same information. The only options are (a) a magic-link / email-driven enrollment that gates registration behind SMTP delivery (not assumed in the self-hosted deployment model), or (b) acceptance of the documented two-request probe. Both are revisited if Ovumcy ever ships a multi-tenant SaaS variant.

A separate vector — replay of a sealed `ovumcy_register_pickup` cookie captured from somebody else's response within the 5-minute TTL — is **not** part of this residual. The pickup cookie carries an opaque nonce that is consumed atomically through `register_pickup_tokens` on the first `GET /register/welcome` call; a captured cookie reaching the welcome endpoint a second time gets the same neutral `/login` redirect as a decoy or expired pickup, and cannot mint a second `ovumcy_auth` session.

## Field-Level Encryption

Sensitive per-account fields written through `security.EncryptField` (currently `users.totp_secret`) are encrypted with AES-256-GCM under a key derived from `SECRET_KEY` via HKDF-SHA256, and bound to the owner's user id through the AEAD authentication tag (additional-authenticated-data, `ovumcy.field.<purpose>:<row id>`). An attacker who can write directly to the database — for example via a hypothetical SQL injection or via host-level compromise — cannot move a ciphertext from one account into another account's row and have it open: the authentication tag fails to verify under a different aad. The `SECRET_KEY` itself is required for both encrypt and decrypt; database access without the key leaks nothing about the persisted value.

A legacy decrypt path exists for ciphertexts written by Ovumcy versions before aad binding was introduced. The new code transparently re-encrypts those values under the aad-bound format on the next successful 2FA login. Operators do not need to take any explicit migration step; the upgrade happens lazily and without revoking the user's current session.

### Login: `requires_totp` reveals 2FA status

`POST /api/auth/login` returns `{"requires_totp": true}` when the supplied password is correct and the account has TOTP enabled, and `{"ok": true}` plus a session cookie otherwise. A credential-dump attacker can use this to triage accounts by whether they have a second factor.

This is inherent to any password-then-TOTP flow that gates the second factor on account state — any uniform response that hides the difference either silently grants access without verifying TOTP (downgrade attack) or unconditionally rejects accounts without TOTP (lockout). The per-account rate limiter (`AuthAttemptPolicy`) and the recovery-code re-auth requirements bound the value of knowing the 2FA status.

## Session Invalidation on Credential Rotation

Operations that rotate a long-lived credential bump `users.auth_session_version` in the same database update, immediately invalidating every active `ovumcy_auth` cookie for that account. This applies to:

- Password change (`POST /api/settings/change-password`).
- Password reset via recovery code (`POST /api/auth/reset-password`).
- Recovery-code regeneration (`POST /api/settings/regenerate-recovery-code`) — the current request receives a freshly issued cookie so the originating session stays alive, but every other device is signed out.
- Forced password reset via the `ovumcy reset-password` operator command.
- TOTP 2FA enable (`POST /api/settings/2fa/verify`) and disable (`POST /api/settings/2fa/disable`) — toggling the second factor is also a change to the account's auth posture, so any cookie issued before the toggle is invalidated. The originating device receives a freshly issued cookie inline; every other device is signed out.

If you suspect a session compromise, regenerating the recovery code is the fastest way to force every other device to re-authenticate without changing your password.

## Cookies

Ovumcy uses only first-party cookies. Cookies marked **Sealed** are encrypted with AES-256-GCM under a key derived from `SECRET_KEY` via HKDF-SHA256 and bound to the cookie name through the AEAD authentication tag, so a value from one cookie cannot be reused as another. `Secure` defaults to the operator's `COOKIE_SECURE` setting unless noted otherwise.

| Name | Purpose | TTL | Path | HttpOnly | SameSite | Secure | Sealed |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ovumcy_auth` | Session token carrying user id, role, and `auth_session_version` | 7 days with remember-me, otherwise browser session | `/` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_csrf` | CSRF double-submit token matched against the `csrf_token` form field on every state-changing request | browser session | `/` | yes | Lax | `COOKIE_SECURE` | no |
| `ovumcy_lang` | User-selected UI language code | 1 year | `/` | no (JS-readable for i18n) | Lax | `COOKIE_SECURE` | no |
| `ovumcy_tz` | Client IANA timezone for server-side calendar math | 1 year | `/` | no (JS-writable, server-validated) | Lax | `COOKIE_SECURE` | no |
| `ovumcy_flash` | One-shot UI message between redirects | 5 minutes | `/` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_recovery_code` | Carries the freshly issued recovery code to `/recovery-code` | 20 minutes | `/` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_register_pickup` | Single-use opaque nonce + decoy recovery code for post-register pickup; consumed atomically through `register_pickup_tokens` | 5 minutes | `/` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_reset_password` | Carries a password-reset token between the recovery-code page and `/reset-password` | 30 minutes | `/` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_oidc_auth` | OIDC `state`, `nonce`, PKCE verifier during sign-in | 10 minutes | `/auth/oidc/callback` | yes | None | forced `true` | yes |
| `ovumcy_oidc_stepup` | OIDC step-up state (purpose, pending password hash) during local-password setup re-auth | 10 minutes | `/auth/oidc/callback` | yes | None | forced `true` | yes |
| `ovumcy_oidc_logout_bridge` | Carries the session id from `/logout` to the provider end-session bridge | 1 minute | `/auth/oidc/logout` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_totp_pending` | 2FA challenge state (user id, remember-me) between password and TOTP submission | 5 minutes (payload-bound) | `/` | yes | Lax | `COOKIE_SECURE` | yes |
| `ovumcy_totp_setup` | Raw TOTP secret transport during enrollment, before the user has confirmed their first code | 5 minutes (payload-bound) | `/` | yes | Lax | `COOKIE_SECURE` | yes |

Notes:

- The two OIDC sign-in cookies require `Secure=true` unconditionally because their `SameSite=None` value is only legal over HTTPS; the OIDC handler refuses to issue them when `COOKIE_SECURE=false`.
- `ovumcy_lang` and `ovumcy_tz` are deliberately non-`HttpOnly` so the browser can write the user's timezone preference and reflect language without a server round trip. They contain no secrets and are validated server-side before use.
- `ovumcy_csrf` is managed by Fiber's CSRF middleware. The OIDC callback path is exempt because it is form-posted from the identity provider, which cannot supply our token; provider-issued `state`/`nonce` cover replay protection for that endpoint instead.

## Data Inventory

What Ovumcy persists per account and per record. All storage is in the operator's configured SQLite file or Postgres database; nothing is sent to any external service.

**`users`** — one row per account:

- Identity: `id`, `email` (unique), `display_name`, `role` (default `owner`), `created_at`.
- Credentials: `password_hash` (bcrypt cost 10), `recovery_code_hash` (bcrypt cost 10), `local_auth_enabled`, `auth_session_version`, `must_change_password`.
- Onboarding: `onboarding_completed`.
- Cycle preferences: `cycle_length`, `period_length`, `luteal_phase`, `auto_period_fill`, `irregular_cycle`, `unpredictable_cycle`, `age_group`, `usage_goal`, `last_period_start`, `long_period_warning_cycle_start`.
- Tracking preferences: `track_bbt`, `temperature_unit`, `track_cervical_mucus`, `hide_sex_chip`, `hide_cycle_factors`, `hide_notes_field`, `show_historical_phases`, `shown_period_tip`.
- 2FA: `totp_enabled`, `totp_secret` (AES-256-GCM aad-bound under an HKDF-derived key, see *Field-Level Encryption*), `totp_last_used_step` (RFC 6238 replay floor).

**`daily_logs`** — one row per (user, calendar day). Dates are stored as UTC midnight and rendered in the user's timezone at read time.

- Period: `is_period`, `cycle_start`, `is_uncertain`, `flow` (`none|spotting|light|medium|heavy`).
- Wellbeing: `mood` (signed scale), `sex_activity` (`none|protected|unprotected`), `bbt` (float, unit selected per account), `cervical_mucus` (`none|dry|moist|creamy|eggwhite`).
- `cycle_factor_keys` (JSON list), `symptom_ids` (JSON list, references owner-managed symptoms), `notes` (free text).

**`symptom_types`** — owner-managed symptom catalog with archive support: `name`, `icon`, `color`, `is_builtin`, soft-archive flag.

**`oidc_identities`** — populated only when OIDC is enabled. `(issuer, subject) → user_id` link plus `created_at`, `last_used_at`. Rows carry `FOREIGN KEY ... ON DELETE CASCADE` on `users.id`.

**`oidc_logout_states`** — short-lived per-session OIDC end-session metadata (`end_session_endpoint`, `id_token_hint`, `post_logout_redirect_url`). Keyed by `session_id` with `expires_at`. Rows are TTL-bounded, not user-id-bounded.

**`register_pickup_tokens`** — opaque single-use nonces for the post-register pickup flow. 5-minute TTL. Rows are not foreign-keyed; they expire and become unreachable on their own.

**Not stored**: analytics, telemetry, third-party identifiers, advertising attribution, error reports, or per-action audit history. Per-action security-event logging is **off by default** and can be toggled per deployment via `AUDIT_LOG_ENABLED` — see *Logging Policy* below.

## Retention and Deletion

**`POST /api/settings/clear-data`** (`Settings → Clear data`) wipes per-account health data while keeping the account active:

- Deletes every `daily_logs` row for the user.
- Deletes every user-defined row in `symptom_types` (built-in symptoms remain).
- Resets cycle and tracking preferences to documented defaults.
- Atomically bumps `auth_session_version`, invalidating every other auth cookie for the account. The originating device is re-issued a fresh cookie inline so the user stays signed in there.

`clear-data` does **not** touch email, password hash, recovery code hash, role, display name, OIDC identity links, TOTP state, or onboarding status.

**`DELETE /api/settings/delete-account`** removes the account entirely:

- Deletes every `daily_logs` row for the user.
- Deletes every `symptom_types` row for the user (including built-ins).
- Deletes the `users` row itself, which cascades to `oidc_identities` via the `ON DELETE CASCADE` foreign key.

Auxiliary short-lived tables (`register_pickup_tokens`, `oidc_logout_states`) are not joined to `users.id`. They expire on their own TTL (≤5 minutes for pickup, provider-driven for logout state); any rows referencing the deleted account become unreachable and are pruned in the normal sweep.

Both operations require the current password through `validateSettingsActionPassword`. OIDC-only accounts must enrol a local password through the step-up re-auth flow before either danger-zone action becomes available.

## Password & Auth Policy

**Local passwords:**

- Minimum length: 8 Unicode code points.
- Required character classes: at least one uppercase letter, one lowercase letter, and one digit (`ValidatePasswordStrength` in `internal/services/password_policy.go`).
- Storage: bcrypt with `bcrypt.DefaultCost` (currently cost 10) via `golang.org/x/crypto/bcrypt`. Hashes live in `users.password_hash`.

**Recovery codes:**

- Shape: `OVUM-XXXX-XXXX-XXXX`, 12 hex characters generated from 48 bits of `crypto/rand` (≈48 bits of effective entropy).
- Storage: bcrypt-hashed in `users.recovery_code_hash`. The plaintext is shown to the user exactly once at issuance and is never retrievable server-side afterwards.
- Online guessing is bounded by the per-account rate limiter (`Rate Limits`, below); the bcrypt cost bounds offline guessing if the database and `SECRET_KEY` leak together.

**Session tokens:**

- `ovumcy_auth` payload is sealed (AES-256-GCM under an HKDF-derived key, see *Cookies* above) and verified per request against `users.auth_session_version`.
- Issued by `setAuthCookie`; reissued inline on the originating device whenever `auth_session_version` is bumped (see *Session Invalidation on Credential Rotation*).

**TOTP 2FA:**

- RFC 6238 with a 30-second step.
- Secrets are AES-256-GCM encrypted at rest with per-row AAD binding (see *Field-Level Encryption*).
- Replay protection: `users.totp_last_used_step` carries the RFC 6238 step index of the last successfully consumed code. `ClaimTOTPStep` performs an atomic `UPDATE … WHERE totp_last_used_step < ?`, so the same code cannot be consumed twice and concurrent submissions of the same step collapse to a single winner.

## Rate Limits

Per-IP HTTP rate limits enforced by Fiber's limiter middleware. Defaults are tunable through environment variables:

| Endpoint | Default budget | Env override |
| --- | --- | --- |
| `POST /api/auth/login` | 8 requests / 15 minutes | `RATE_LIMIT_LOGIN_MAX`, `RATE_LIMIT_LOGIN_WINDOW` |
| `POST /api/auth/register` | 8 requests / 15 minutes | `RATE_LIMIT_REGISTER_MAX`, `RATE_LIMIT_REGISTER_WINDOW` |
| `POST /api/auth/forgot-password` | 8 requests / 1 hour | `RATE_LIMIT_FORGOT_PASSWORD_MAX`, `RATE_LIMIT_FORGOT_PASSWORD_WINDOW` |
| `POST /api/auth/logout` | 60 requests / 15 minutes | `RATE_LIMIT_LOGOUT_MAX`, `RATE_LIMIT_LOGOUT_WINDOW` |
| `/api/*` (catch-all) | 300 requests / 1 minute | `RATE_LIMIT_API_MAX`, `RATE_LIMIT_API_WINDOW` |

Plus per-account, identity-keyed budgets enforced by `AuthAttemptPolicy` (`internal/services/auth_attempt_policy.go`):

- Login attempts: 8 failures / 15 minutes.
- Logout attempts: 20 failures / 15 minutes (account-scoped).
- TOTP login challenge: 5 failures / 15 minutes.
- TOTP disable: 5 failures / 15 minutes.

Per-account budgets are keyed by `HMAC-SHA256(SECRET_KEY, "ovumcy.auth-attempt.identity.v1:" || identity)`, so the limiter never persists the raw identifier.

## SECRET_KEY Usage Map

`SECRET_KEY` (or the file behind `SECRET_KEY_FILE`) is the single application-wide secret. It is used as the input keying material for several distinct subsystems:

| Subsystem | Derivation | Effect of rotation |
| --- | --- | --- |
| Sealed cookies (every `Sealed=yes` cookie above) | `HKDF-SHA256(SECRET_KEY, salt="ovumcy.secure-cookie.salt.v2", info="ovumcy.secure-cookie.key.v2")` → AES-256-GCM, with cookie name as AAD | Existing cookies fail to open and are silently re-issued or rejected. Users see a fresh sign-in. |
| Field encryption (`users.totp_secret`) | `HKDF-SHA256(SECRET_KEY, salt="ovumcy.field-crypto.salt.v1", info="ovumcy.field-crypto.key.v1")` → AES-256-GCM, with `ovumcy.field.<purpose>:<row id>` as AAD | **Existing TOTP ciphertexts no longer decrypt.** All 2FA-enabled accounts will fail their next TOTP challenge — see the caveat below. |
| Rate-limit identity HMAC | `HMAC-SHA256(SECRET_KEY, "ovumcy.auth-attempt.identity.v1:" || identity)` | Existing per-identity counters become unreachable; cooldowns reset. Self-recovering. |
| Auth session token signing | Used inside `BuildAuthSessionTokenWithSessionID` to sign session tokens before they are sealed into `ovumcy_auth` | Subsumed by the sealed-cookie invalidation row above. |

**Operational caveat — rotating `SECRET_KEY` with 2FA accounts:**

`users.totp_secret` is encrypted under a key derived from the current `SECRET_KEY`. Rotating the secret without a coordinated re-encryption step leaves TOTP-enabled accounts unable to complete the 2FA challenge — they will see a failed verification even when their authenticator app produces the correct code. Recovery options for each affected user are:

1. Sign in with the recovery code, then re-enrol TOTP under the new key.
2. Ask the operator to run `ovumcy reset-password <email>` and disable 2FA via Settings once signed back in.

Plan secret rotation as planned maintenance, communicate it in advance, and consider asking users to disable 2FA before the rotation window.

## Threat Model

**In scope** — Ovumcy actively defends against:

- Credential stuffing and bot enumeration against `/api/auth/*` (rate limits, sealed pickup, bcrypt-timing equalization).
- Replay of captured sealed cookies (server-side single-use for register pickup; session-version checks for `ovumcy_auth`).
- DOM-XSS into HTMX error responses (status-error fragment is parsed with `DOMParser` and rebuilt via `document.createElement` + `textContent`, never `innerHTML`).
- Cross-site form submission (CSRF middleware on every state-changing endpoint; OIDC callback uses provider-issued `state`/`nonce` instead).
- Algorithm-confusion attacks against ID tokens (asymmetric-algorithm allowlist; `HS*` and `none` are rejected even if the provider advertises them).
- Malicious OIDC discovery metadata redirecting the logout flow to attacker-controlled hosts (`end_session_endpoint` host-pinned to the configured issuer).
- Cross-account ciphertext substitution at the database layer (field encryption is AAD-bound to the row id).
- Trivial password reuse (8-character minimum with three required character classes).

**Out of scope** — Ovumcy assumes the operator is trusted, and does not defend against:

- An operator who reads the SQLite file directly, captures `SECRET_KEY`, or inspects process memory. The deployment model is single-tenant self-hosting; in the typical case the operator is the same person as the user.
- Compromise of the user's endpoint (browser malware, OS keyloggers, shoulder surfing).
- TLS downgrade attacks against an operator who runs the app over plain HTTP and ignores the `COOKIE_SECURE` advisory.
- Side-channel attacks against the host CPU (Spectre-class, cache-timing); Ovumcy uses constant-time comparisons where it matters but cannot defend the underlying hardware.
- Simultaneous compromise of `SECRET_KEY` and the OIDC provider's signing key.

Multi-tenant SaaS deployment, organizational identity management, hardware-key support, and audit-log compliance are not goals of this codebase. They are explicitly out of scope for both threat modeling and feature roadmap.

## Logging Policy

Ovumcy does **not** emit per-action audit logs by default. The `AUDIT_LOG_ENABLED` environment variable controls the audit-event stream:

- `AUDIT_LOG_ENABLED=false` (default) — the runtime emits no `security event:` lines. Go panics, startup configuration errors, and the Fiber request log remain enabled.
- `AUDIT_LOG_ENABLED=true` — the runtime emits per-action security-event lines to stderr via the Go standard `log` package. Each line includes the action name, outcome, request method, **sanitized** request path (concrete date parameters are replaced with `:date` and other identifiers are similarly masked), response format, and — for authenticated requests — `user_id` and role. Example:

  ```
  security event: action="health.day_upsert" outcome="success" method="POST"
                  path="/api/days/:date" format="json" user_id="42"
                  role="owner" domain="health_data" target="day_entry"
  ```

When enabled, these lines are visible to the operator through their container runtime (`docker compose logs`, journald, etc.) and never leave the host. They are intended for ad-hoc incident investigation — for example, to confirm whether a suspected compromise produced state-changing requests, and from which `user_id`. The audit stream is not designed as a compliance audit trail; nothing in Ovumcy itself ships, archives, or rotates these lines.

If you enable `AUDIT_LOG_ENABLED=true`, plan retention and access control around the persistent-identifier content (`user_id`, role). Treat the resulting log stream as the same sensitivity class as the database itself.

The Fiber request log (`time | status | latency | method | path`) is independent of `AUDIT_LOG_ENABLED` and remains enabled in all configurations. It does not include `user_id` or authenticated-session metadata.

The startup banner reflects the current setting (`audit_log=true|false`) so operators can confirm the effective configuration on each boot.
