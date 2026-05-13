package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ovumcy/ovumcy-web/internal/models"
	"github.com/ovumcy/ovumcy-web/internal/services"
)

func TestCalendarDayPanelReadonlySummaryShowsSavedBBT(t *testing.T) {
	app, database := newOnboardingTestApp(t)
	user := createOnboardingTestUser(t, database, "calendar-bbt-summary@example.com", "StrongPass1", true)
	if err := database.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"track_bbt":        true,
		"temperature_unit": services.TemperatureUnitCelsius,
	}).Error; err != nil {
		t.Fatalf("enable BBT tracking: %v", err)
	}

	logEntry := models.DailyLog{
		UserID: user.ID,
		Date:   time.Date(2026, time.February, 17, 0, 0, 0, 0, time.UTC),
		BBT:    36.75,
		Notes:  "tracked",
	}
	if err := database.Create(&logEntry).Error; err != nil {
		t.Fatalf("create daily log: %v", err)
	}

	authCookie := loginAndExtractAuthCookie(t, app, user.Email, "StrongPass1")
	request := httptest.NewRequest(http.MethodGet, "/calendar/day/2026-02-17", nil)
	request.Header.Set("Accept-Language", "en")
	request.Header.Set("Cookie", authCookie)

	response := mustAppResponse(t, app, request)
	assertStatusCode(t, response, http.StatusOK)

	rendered := mustReadBodyString(t, response.Body)
	if !strings.Contains(rendered, "BBT") {
		t.Fatalf("expected BBT label in calendar day summary, got %q", rendered)
	}
	if !strings.Contains(rendered, "36.75 °C") {
		t.Fatalf("expected saved BBT value in calendar day summary, got %q", rendered)
	}
}

func TestCalendarDayPanelEditModeRendersDeleteActionForExistingEntry(t *testing.T) {
	app, database := newOnboardingTestApp(t)
	user := createOnboardingTestUser(t, database, "calendar-confirm@example.com", "StrongPass1", true)
	authCookie := loginAndExtractAuthCookie(t, app, user.Email, "StrongPass1")

	logEntry := models.DailyLog{
		UserID:   user.ID,
		Date:     time.Date(2026, time.February, 17, 0, 0, 0, 0, time.UTC),
		IsPeriod: true,
		Flow:     models.FlowMedium,
		Notes:    "entry",
	}
	if err := database.Create(&logEntry).Error; err != nil {
		t.Fatalf("create daily log: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/calendar/day/2026-02-17?mode=edit", nil)
	request.Header.Set("Accept-Language", "en")
	request.Header.Set("Cookie", authCookie)

	response, err := app.Test(request, -1)
	if err != nil {
		t.Fatalf("calendar day panel request failed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read panel body: %v", err)
	}
	rendered := string(body)

	assertBodyContainsAll(t, rendered,
		bodyStringMatch{fragment: `data-day-delete-form`, message: "expected delete form affordance for existing calendar entry"},
		bodyStringMatch{fragment: `data-day-delete-button`, message: "expected delete button affordance for existing calendar entry"},
	)
}

func TestCalendarDayPanelEditModePreservesAndSavesPeriodToggle(t *testing.T) {
	app, database := newOnboardingTestApp(t)
	user := createOnboardingTestUser(t, database, "calendar-period-toggle@example.com", "StrongPass1", true)
	authCookie := loginAndExtractAuthCookie(t, app, user.Email, "StrongPass1")

	day := time.Date(2026, time.February, 17, 0, 0, 0, 0, time.UTC)
	logEntry := models.DailyLog{
		UserID:   user.ID,
		Date:     day,
		IsPeriod: true,
		Flow:     models.FlowLight,
		Notes:    "entry",
	}
	if err := database.Create(&logEntry).Error; err != nil {
		t.Fatalf("create daily log: %v", err)
	}

	panelRequest := httptest.NewRequest(http.MethodGet, "/calendar/day/2026-02-17?mode=edit", nil)
	panelRequest.Header.Set("Accept-Language", "en")
	panelRequest.Header.Set("Cookie", authCookie)

	panelResponse, err := app.Test(panelRequest, -1)
	if err != nil {
		t.Fatalf("calendar day panel request failed: %v", err)
	}
	defer panelResponse.Body.Close()

	if panelResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", panelResponse.StatusCode)
	}

	panelBody, err := io.ReadAll(panelResponse.Body)
	if err != nil {
		t.Fatalf("read panel body: %v", err)
	}
	checkedPattern := regexp.MustCompile(`(?s)name="is_period"[^>]*checked`)
	if !checkedPattern.Match(panelBody) {
		t.Fatalf("expected edit-mode period toggle to stay checked for persisted period log")
	}

	form := url.Values{
		"flow":  {models.FlowNone},
		"notes": {"updated"},
	}
	saveRequest := httptest.NewRequest(http.MethodPost, "/api/days/2026-02-17", strings.NewReader(form.Encode()))
	saveRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	saveRequest.Header.Set("HX-Request", "true")
	saveRequest.Header.Set("Accept-Language", "en")
	saveRequest.Header.Set("Cookie", authCookie)

	saveResponse, err := app.Test(saveRequest, -1)
	if err != nil {
		t.Fatalf("save request failed: %v", err)
	}
	defer saveResponse.Body.Close()

	if saveResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected save status 200, got %d", saveResponse.StatusCode)
	}

	var updated models.DailyLog
	if err := database.Where("user_id = ? AND date = ?", user.ID, day).First(&updated).Error; err != nil {
		t.Fatalf("load updated log: %v", err)
	}
	if updated.IsPeriod {
		t.Fatalf("expected unchecked edit-mode period toggle to persist as false")
	}
}
