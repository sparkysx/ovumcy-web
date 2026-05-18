import { test, expect, type Page } from '@playwright/test';
import {
  continueFromRecoveryCode,
  createCredentials,
  expectInlineRegisterRecoveryStep,
  readRecoveryCode,
  registerOwnerViaUI,
} from './support/auth-helpers';
import { dateFieldRoot, fillDateField } from './support/date-field-helpers';
import { setRequestTimezoneFromBrowser } from './support/timezone-helpers';
import { todayISOFromDashboard } from './support/stats-helpers';

function isoDateDaysAgo(days: number): string {
  const date = new Date();
  date.setHours(0, 0, 0, 0);
  date.setDate(date.getDate() - days);
  const yyyy = date.getFullYear();
  const mm = String(date.getMonth() + 1).padStart(2, '0');
  const dd = String(date.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

async function registerAndOnboardWithStartDaysAgo(
  page: Page,
  prefix: string,
  startDaysAgo: number,
): Promise<void> {
  // completeOnboardingIfPresent hardcodes today-3 as the period start, which
  // makes today an auto-period-fill day. Spotting-warning + future-cycle-
  // start scenarios need today to sit outside the onboarding period cluster,
  // so we run a custom onboarding flow that submits an explicit older date.
  const credentials = createCredentials(prefix);
  await registerOwnerViaUI(page, credentials);
  await expectInlineRegisterRecoveryStep(page);
  await readRecoveryCode(page);
  await continueFromRecoveryCode(page);

  const startISO = isoDateDaysAgo(startDaysAgo);
  const startInput = page.locator('#last-period-start');
  await expect(dateFieldRoot(startInput)).toBeVisible();
  await fillDateField(startInput, startISO);
  await page.locator('form[hx-post="/api/v1/onboarding/steps/1"] button[type="submit"]').click();

  const stepTwoForm = page.locator('form[hx-post="/api/v1/onboarding/steps/2"]');
  await expect(stepTwoForm).toBeVisible();
  await Promise.all([
    page.waitForURL(/\/dashboard(?:\?.*)?$/, { timeout: 15000 }),
    stepTwoForm.locator('button[type="submit"]').click(),
  ]);

  await setRequestTimezoneFromBrowser(page);
}

async function csrfToken(page: Page): Promise<string> {
  return (await page.locator('meta[name="csrf-token"]').getAttribute('content')) ?? '';
}

test.describe('Dashboard: spotting cycle warning', () => {
  test('saving today as a period day with spotting flow surfaces the day-1 spotting tip on the dashboard', async ({
    page,
  }) => {
    // Anchor onboarding 30 days back so the auto-period-fill window
    // (today-30 .. today-26) sits well before today. currentPeriodStreak
    // AtDay walks backwards from today: with no period days adjacent, the
    // streak collapses to 1 and cycleStart = today, which is exactly what
    // shouldShowSpottingCycleWarning needs.
    await registerAndOnboardWithStartDaysAgo(page, 'dashboard-spotting-warning', 30);
    const today = await todayISOFromDashboard(page);

    const response = await page.request.put(`/api/v1/days/${today}`, {
      headers: {
        'X-CSRF-Token': await csrfToken(page),
        'Content-Type': 'application/json',
      },
      data: { is_period: true, flow: 'spotting' },
    });
    expect(response.status()).toBeLessThan(400);

    await page.goto('/dashboard');
    await expect(page).toHaveURL(/\/dashboard$/);

    // The warning sits inside the [data-period-fields] fieldset right after
    // the flow chips — scope the locator so a generic copy match on
    // .journal-muted elsewhere on the page cannot mask a regression.
    const periodFields = page.locator('[data-period-fields]');
    await expect(periodFields).toBeVisible();
    await expect(periodFields).toContainText('Spotting may not be day 1. Check again tomorrow.');
  });
});
