package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovumcy/ovumcy-web/internal/models"
	"github.com/ovumcy/ovumcy-web/internal/services"
)

func TestDayReadEndpointsUseRequestTimezoneForLocalCalendarDay(t *testing.T) {
	app, database, _ := newOnboardingTestAppWithLocation(t, time.UTC)
	user := createOnboardingTestUser(t, database, "days-read-timezone@example.com", "StrongPass1", true)
	authCookie := loginAndExtractAuthCookie(t, app, user.Email, "StrongPass1")

	nowUTC := time.Now().UTC()
	timezoneName, location := timezoneWithDifferentCalendarDay(t, nowUTC)
	localDay := services.DateAtLocation(nowUTC.In(location), location)
	localDayRaw := localDay.Format("2006-01-02")

	seed := models.DailyLog{
		UserID:   user.ID,
		Date:     localDay,
		IsPeriod: true,
		Flow:     models.FlowLight,
		Notes:    "timezone read note",
	}
	if err := database.Create(&seed).Error; err != nil {
		t.Fatalf("seed timezone daily log: %v", err)
	}

	cookieHeader := joinCookieHeader(authCookie, timezoneCookieName+"="+timezoneName)

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/days?from="+localDayRaw+"&to="+localDayRaw, nil)
	listRequest.Header.Set("Accept", "application/json")
	listRequest.Header.Set("Cookie", cookieHeader)
	listRequest.Header.Set(timezoneHeaderName, timezoneName)

	listResponse := mustAppResponse(t, app, listRequest)
	assertStatusCode(t, listResponse, http.StatusOK)

	var listPayload []models.DailyLog
	if err := json.NewDecoder(listResponse.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode day list payload: %v", err)
	}
	if len(listPayload) != 1 {
		t.Fatalf("expected one log for request-local day %s, got %#v", localDayRaw, listPayload)
	}
	if listPayload[0].Notes != seed.Notes {
		t.Fatalf("expected listed notes %q, got %q", seed.Notes, listPayload[0].Notes)
	}

	dayRequest := httptest.NewRequest(http.MethodGet, "/api/v1/days/"+localDayRaw, nil)
	dayRequest.Header.Set("Accept", "application/json")
	dayRequest.Header.Set("Cookie", cookieHeader)
	dayRequest.Header.Set(timezoneHeaderName, timezoneName)

	dayResponse := mustAppResponse(t, app, dayRequest)
	assertStatusCode(t, dayResponse, http.StatusOK)

	dayPayload := models.DailyLog{}
	if err := json.NewDecoder(dayResponse.Body).Decode(&dayPayload); err != nil {
		t.Fatalf("decode day payload: %v", err)
	}
	if dayPayload.Notes != seed.Notes {
		t.Fatalf("expected fetched day notes %q, got %q", seed.Notes, dayPayload.Notes)
	}

	existsRequest := httptest.NewRequest(http.MethodHead, "/api/v1/days/"+localDayRaw, nil)
	existsRequest.Header.Set("Cookie", cookieHeader)
	existsRequest.Header.Set(timezoneHeaderName, timezoneName)

	existsResponse := mustAppResponse(t, app, existsRequest)
	assertStatusCode(t, existsResponse, http.StatusOK)
}
