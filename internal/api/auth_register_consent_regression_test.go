package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestRegisterRejectsMissingConsent(t *testing.T) {
	app, _ := newOnboardingTestApp(t)

	form := url.Values{
		"email":            {"missing-consent@example.com"},
		"password":         {"StrongPass1"},
		"confirm_password": {"StrongPass1"},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response := mustAppResponse(t, app, request)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing consent, got %d", response.StatusCode)
	}

	body := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	detail, _ := body["error_detail"].(map[string]any)
	if detail == nil {
		t.Fatalf("expected error_detail object, got %v", body)
	}
	if got, _ := detail["key"].(string); got != "consent required" {
		t.Fatalf("expected error_detail.key %q, got %q", "consent required", got)
	}
}

func TestRegisterRejectsExplicitlyFalseConsent(t *testing.T) {
	app, _ := newOnboardingTestApp(t)

	form := url.Values{
		"email":            {"false-consent@example.com"},
		"password":         {"StrongPass1"},
		"confirm_password": {"StrongPass1"},
		"consent":          {"false"},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response := mustAppResponse(t, app, request)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for explicit consent=false, got %d", response.StatusCode)
	}
}

func TestRegisterPageRendersConsentControl(t *testing.T) {
	app, _ := newOnboardingTestApp(t)

	request := httptest.NewRequest(http.MethodGet, "/register", nil)
	request.Header.Set("Accept-Language", "en")

	response := mustAppResponse(t, app, request)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected register page status 200, got %d", response.StatusCode)
	}

	document := mustParseHTMLDocument(t, mustReadBodyString(t, response.Body))
	consent := htmlFindElement(document, func(node *html.Node) bool {
		return node.Type == html.ElementNode && htmlHasAttr(node, "data-register-consent")
	})
	if consent == nil {
		t.Fatal("expected register page to render the GDPR consent control")
	}
	checkbox := htmlFindElement(consent, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "input" && htmlAttr(node, "name") == "consent"
	})
	if checkbox == nil {
		t.Fatal("expected consent control to contain an <input name=\"consent\">")
	}
	if !htmlHasAttr(checkbox, "required") {
		t.Fatal("expected consent checkbox to carry the HTML required attribute")
	}
}

func TestRegisterFlashCarriesConsentRequiredKeyOnBrowserRedirect(t *testing.T) {
	app, _ := newOnboardingTestApp(t)

	form := url.Values{
		"email":            {"redirect-consent@example.com"},
		"password":         {"StrongPass1"},
		"confirm_password": {"StrongPass1"},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := mustAppResponse(t, app, request)
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected browser redirect (303) for missing consent, got %d", response.StatusCode)
	}
	if location := response.Header.Get("Location"); location != "/register" {
		t.Fatalf("expected redirect to /register, got %q", location)
	}

	flashValue := responseCookieValue(response.Cookies(), flashCookieName)
	if flashValue == "" {
		t.Fatal("expected flash cookie carrying consent_required error key")
	}

	followRequest := httptest.NewRequest(http.MethodGet, "/register", nil)
	followRequest.Header.Set("Accept-Language", "en")
	followRequest.Header.Set("Cookie", flashCookieName+"="+flashValue)
	followResponse := mustAppResponse(t, app, followRequest)
	if followResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected follow-up register status 200, got %d", followResponse.StatusCode)
	}

	document := mustParseHTMLDocument(t, mustReadBodyString(t, followResponse.Body))
	if htmlAuthErrorByKey(document, "auth.error.consent_required") == nil {
		t.Fatal("expected follow-up register page to render auth.error.consent_required banner")
	}
}
