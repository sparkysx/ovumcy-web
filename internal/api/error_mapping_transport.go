package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ovumcy/ovumcy-web/internal/httpx"
	"github.com/ovumcy/ovumcy-web/internal/services"
)

// apiError renders an error response in the format matching the request.
// HTML/HTMX requests get a localized status fragment; JSON requests get the
// standard envelope with both the legacy top-level `error` string key and the
// richer `error_detail` object describing category and target. The top-level
// key stays for backward compatibility with clients that already parse it.
func apiError(c *fiber.Ctx, spec APIErrorSpec) error {
	if responseFormat(c) == httpx.ResponseFormatHTMX {
		rendered := spec.Key
		if key := services.AuthErrorTranslationKey(spec.Key); key != "" {
			if localized := translateMessage(currentMessages(c), key); localized != key {
				rendered = localized
			}
		} else if localized := translateMessage(currentMessages(c), spec.Key); localized != spec.Key {
			rendered = localized
		}
		return c.Status(spec.Status).SendString(httpx.StatusErrorMarkup(rendered))
	}
	return c.Status(spec.Status).JSON(fiber.Map{
		"error": spec.Key,
		"error_detail": fiber.Map{
			"key":      spec.Key,
			"category": string(spec.Category),
			"target":   string(spec.Target),
		},
	})
}

func (handler *Handler) respondAuthError(c *fiber.Ctx, spec APIErrorSpec) error {
	if (strings.HasPrefix(c.Path(), "/api/auth/") || strings.HasPrefix(c.Path(), "/auth/oidc")) && !acceptsJSON(c) && !isHTMX(c) {
		flash := FlashPayload{AuthError: spec.Key}
		switch c.Path() {
		case "/api/auth/register":
			email := services.NormalizeAuthEmail(c.FormValue("email"))
			flash.RegisterEmail = email
			handler.setFlashCookie(c, flash)
			return c.Redirect("/register", fiber.StatusSeeOther)
		case "/api/auth/login":
			flash.LoginEmail = services.NormalizeAuthEmail(c.FormValue("email"))
			handler.setFlashCookie(c, flash)
			return c.Redirect("/login", fiber.StatusSeeOther)
		case "/api/auth/forgot-password":
			flash.ForgotEmail = services.NormalizeAuthEmail(c.FormValue("email"))
			handler.setFlashCookie(c, flash)
			return c.Redirect("/forgot-password", fiber.StatusSeeOther)
		case "/auth/oidc", "/auth/oidc/start", "/auth/oidc/callback":
			handler.setFlashCookie(c, flash)
			return c.Redirect("/login", fiber.StatusSeeOther)
		case "/api/auth/reset-password":
			handler.setFlashCookie(c, flash)
			return c.Redirect("/reset-password", fiber.StatusSeeOther)
		case "/api/auth/2fa":
			handler.setFlashCookie(c, flash)
			return c.Redirect("/auth/2fa", fiber.StatusSeeOther)
		default:
			handler.setFlashCookie(c, flash)
			return c.Redirect("/login", fiber.StatusSeeOther)
		}
	}
	return apiError(c, spec)
}

func (handler *Handler) respondSettingsError(c *fiber.Ctx, spec APIErrorSpec) error {
	if isHTMX(c) {
		rendered := spec.Key
		if key := services.AuthErrorTranslationKey(spec.Key); key != "" {
			if localized := translateMessage(currentMessages(c), key); localized != key {
				rendered = localized
			}
		}
		return c.Status(fiber.StatusOK).SendString(httpx.StatusErrorMarkup(rendered))
	}
	if (strings.HasPrefix(c.Path(), "/api/settings/") || strings.HasPrefix(c.Path(), "/settings/")) && !acceptsJSON(c) {
		handler.setFlashCookie(c, FlashPayload{SettingsError: spec.Key})
		return c.Redirect("/settings", fiber.StatusSeeOther)
	}
	return apiError(c, spec)
}
