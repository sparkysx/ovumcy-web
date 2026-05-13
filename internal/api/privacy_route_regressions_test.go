package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestPrivacyRouteRendersPublicPage(t *testing.T) {
	app := newTestAppWithPrivacyRoute(t)

	request := httptest.NewRequest(http.MethodGet, "/privacy", nil)
	request.Header.Set("Accept-Language", "en")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	rendered := string(body)
	if !strings.Contains(rendered, "Privacy Policy") {
		t.Fatalf("expected rendered page to contain privacy title, got body: %s", rendered)
	}
	if !strings.Contains(rendered, "Zero Data Collection") {
		t.Fatalf("expected rendered page to contain privacy section, got body: %s", rendered)
	}
	if !strings.Contains(rendered, "Hidden sections and exports") {
		t.Fatalf("expected rendered page to explain hidden sections and exports, got body: %s", rendered)
	}
	if !strings.Contains(rendered, "SQLite or PostgreSQL database on this server") {
		t.Fatalf("expected rendered page to describe server-side storage, got body: %s", rendered)
	}
	if strings.Contains(rendered, "Ovumcy is built for private, self-hosted tracking.") {
		t.Fatalf("did not expect deprecated privacy subtitle to be rendered")
	}
	if !strings.Contains(rendered, `href="/login"`) {
		t.Fatalf("expected back link to point to /login for guest users")
	}
}

func TestPrivacyRouteBackLinkForAuthenticatedUser(t *testing.T) {
	app, database := newOnboardingTestApp(t)
	user := createOnboardingTestUser(t, database, "privacy-auth@example.com", "StrongPass1", true)
	authCookie := loginAndExtractAuthCookie(t, app, user.Email, "StrongPass1")

	request := httptest.NewRequest(http.MethodGet, "/privacy", nil)
	request.Header.Set("Cookie", authCookie)
	request.Header.Set("Accept-Language", "en")

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	rendered := string(body)
	if !strings.Contains(rendered, `href="/dashboard"`) {
		t.Fatalf("expected privacy page to include dashboard backlink for authenticated users")
	}
	breadcrumbPattern := regexp.MustCompile(`(?s)<p class="journal-muted text-sm">\s*<a href="/dashboard" class="inline-link">Dashboard</a>`)
	if !breadcrumbPattern.MatchString(rendered) {
		t.Fatalf("expected breadcrumb to use dashboard naming for authenticated users")
	}
}
