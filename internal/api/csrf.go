package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
)

// CSRFTokenExtractor pulls the CSRF token from the request. It accepts the
// legacy `csrf_token` form field (used by browser HTMX submissions) and the
// `X-CSRF-Token` request header (used by non-browser API clients). The form
// field is checked first because it dominates browser traffic; the header is
// the fallback path for programmatic clients that do not send form bodies.
// Returns `csrf.ErrTokenNotFound` when neither source carries a non-blank
// value so that the middleware's ErrorHandler can classify the rejection.
func CSRFTokenExtractor(c *fiber.Ctx) (string, error) {
	if token := strings.TrimSpace(c.FormValue("csrf_token")); token != "" {
		return token, nil
	}
	if token := strings.TrimSpace(c.Get("X-CSRF-Token")); token != "" {
		return token, nil
	}
	return "", csrf.ErrTokenNotFound
}

// CSRFFailureReason maps the error returned by the CSRF middleware into a
// stable, log-safe reason key. Only the documented sentinel errors carry a
// meaningful classification; anything else is collapsed to "csrf rejected"
// so we never leak transport details into the audit log.
func CSRFFailureReason(err error) string {
	switch {
	case errors.Is(err, csrf.ErrTokenInvalid):
		return "invalid token"
	case errors.Is(err, csrf.ErrTokenNotFound):
		return "missing token"
	case errors.Is(err, csrf.ErrNoReferer), errors.Is(err, csrf.ErrBadReferer):
		return "invalid referer"
	default:
		return "csrf rejected"
	}
}
