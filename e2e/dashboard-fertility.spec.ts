import { test, expect } from '@playwright/test';
import {
  completeOnboardingIfPresent,
  continueFromRecoveryCode,
  createCredentials,
  expectInlineRegisterRecoveryStep,
  readRecoveryCode,
  registerOwnerViaUI,
} from './support/auth-helpers';
import { setRequestTimezoneFromBrowser } from './support/timezone-helpers';
import { openCalendarDayEditor, todayISOFromDashboard } from './support/stats-helpers';

test.describe('Dashboard: fertility badge', () => {
  test('eggwhite cervical mucus shows the High fertility badge on dashboard', async ({ page }) => {
    const credentials = createCredentials('fertility-eggwhite');
    await registerOwnerViaUI(page, credentials);
    await expectInlineRegisterRecoveryStep(page);
    await readRecoveryCode(page);
    await continueFromRecoveryCode(page);
    await completeOnboardingIfPresent(page);
    await setRequestTimezoneFromBrowser(page);

    // Cervical mucus tracking is off by default; enable it so the day editor
    // exposes the cervical_mucus radio group.
    await page.goto('/settings');
    await expect(page).toHaveURL(/\/settings$/);
    const trackingSection = page.locator('#settings-tracking');
    const trackCervicalMucus = trackingSection.locator('input[name="track_cervical_mucus"]');
    await trackCervicalMucus.check();
    const trackingForm = trackingSection.locator('form[data-settings-draft-form="tracking"]');
    await trackingForm.evaluate((node) => {
      if (node instanceof HTMLFormElement) {
        node.requestSubmit();
      }
    });
    await expect(page.locator('#settings-tracking-status .status-ok')).toBeVisible();

    // Open today's calendar editor and pick the eggwhite option.
    const today = await todayISOFromDashboard(page);
    const form = await openCalendarDayEditor(page, today);
    // The radio input is visually wrapped by a styled <span class="radio-tile">
    // that intercepts pointer events; click the label so the event still hits
    // the input. Mirrors the pattern used by saveCycleFactorOnDay.
    const eggwhiteLabel = form.locator(
      'label.choice-option:has(input[name="cervical_mucus"][value="eggwhite"])'
    );
    await eggwhiteLabel.click();
    await Promise.all([
      page.waitForResponse((response) => {
        return (
          response.request().method() === 'PUT' &&
          response.url().includes(`/api/v1/days/${today}`)
        );
      }),
      form.evaluate((node) => {
        if (node instanceof HTMLFormElement) {
          node.requestSubmit();
        }
      }),
    ]);

    // Dashboard hero now carries the high-fertility badge. For the default
    // usage_goal=health the localized text is "High fertility" and the badge
    // has neither the warning nor the positive variant class.
    await page.goto('/dashboard');
    await expect(page).toHaveURL(/\/dashboard$/);

    const heroBadge = page.locator('.dashboard-cycle-hero-badge');
    await expect(heroBadge).toBeVisible();
    await expect(heroBadge).toContainText('High fertility');
    await expect(heroBadge).not.toHaveClass(/dashboard-cycle-hero-badge-warning/);
    await expect(heroBadge).not.toHaveClass(/dashboard-cycle-hero-badge-positive/);
    // The fallback `.dashboard-status-item` copy of the badge only renders
    // when the cycle hero is hidden (`{{if not .CycleHero.Visible}}` in
    // dashboard.html) — exercised separately by tests that suppress the hero.
  });
});
