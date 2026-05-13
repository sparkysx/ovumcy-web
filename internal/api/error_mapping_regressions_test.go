package api

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/ovumcy/ovumcy-web/internal/services"
)

func TestCommonErrorSpecs(t *testing.T) {
	testCases := []struct {
		name string
		got  APIErrorSpec
		want APIErrorSpec
	}{
		{
			name: "unauthorized",
			got:  unauthorizedErrorSpec(),
			want: globalErrorSpec(fiber.StatusUnauthorized, APIErrorCategoryUnauthorized, "unauthorized"),
		},
		{
			name: "onboarding required",
			got:  onboardingRequiredErrorSpec(),
			want: globalErrorSpec(fiber.StatusForbidden, APIErrorCategoryForbidden, "onboarding required"),
		},
		{
			name: "owner access required",
			got:  ownerAccessRequiredErrorSpec(),
			want: globalErrorSpec(fiber.StatusForbidden, APIErrorCategoryForbidden, "owner access required"),
		},
		{
			name: "setup state load",
			got:  setupStateLoadErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to load setup state"),
		},
		{
			name: "invalid month",
			got:  invalidMonthErrorSpec(),
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid month"),
		},
		{
			name: "not found",
			got:  notFoundErrorSpec(),
			want: globalErrorSpec(fiber.StatusNotFound, APIErrorCategoryNotFound, "not found"),
		},
		{
			name: "template not found",
			got:  templateNotFoundErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "template not found"),
		},
		{
			name: "template render",
			got:  templateRenderErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to render template"),
		},
		{
			name: "partial not found",
			got:  partialNotFoundErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "partial not found"),
		},
		{
			name: "partial render",
			got:  partialRenderErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to render partial"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.got != testCase.want {
				t.Fatalf("unexpected mapped error: got %#v want %#v", testCase.got, testCase.want)
			}
		})
	}
}

func TestMapExportRangeError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want APIErrorSpec
	}{
		{
			name: "invalid from",
			err:  services.ErrExportFromDateInvalid,
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid from date"),
		},
		{
			name: "invalid to",
			err:  services.ErrExportToDateInvalid,
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid to date"),
		},
		{
			name: "invalid range",
			err:  services.ErrExportRangeInvalid,
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid range"),
		},
		{
			name: "unknown",
			err:  errors.New("unknown"),
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid range"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if got := mapExportRangeError(testCase.err); got != testCase.want {
				t.Fatalf("unexpected mapped error: got %#v want %#v", got, testCase.want)
			}
		})
	}
}

func TestOnboardingErrorSpecs(t *testing.T) {
	testCases := []struct {
		name string
		got  APIErrorSpec
		want APIErrorSpec
	}{
		{
			name: "validation",
			got:  onboardingValidationErrorSpec("invalid input"),
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid input"),
		},
		{
			name: "save step",
			got:  onboardingSaveStepErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to save onboarding step"),
		},
		{
			name: "steps required",
			got:  onboardingStepsRequiredErrorSpec(),
			want: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "complete onboarding steps first"),
		},
		{
			name: "finish",
			got:  onboardingFinishErrorSpec(),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to finish onboarding"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.got != testCase.want {
				t.Fatalf("unexpected mapped error: got %#v want %#v", testCase.got, testCase.want)
			}
		})
	}
}

func TestMapSettingsPasswordChangeError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want APIErrorSpec
	}{
		{
			name: "invalid input",
			err:  services.ErrSettingsPasswordChangeInvalidInput,
			want: settingsFormErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid settings input"),
		},
		{
			name: "password mismatch",
			err:  services.ErrSettingsPasswordMismatch,
			want: settingsFormErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "password mismatch"),
		},
		{
			name: "invalid current password",
			err:  services.ErrSettingsInvalidCurrentPassword,
			want: settingsFormErrorSpec(fiber.StatusUnauthorized, APIErrorCategoryUnauthorized, "invalid current password"),
		},
		{
			name: "local password required",
			err:  services.ErrSettingsLocalPasswordNotSet,
			want: settingsLocalPasswordRequiredErrorSpec(),
		},
		{
			name: "new password must differ",
			err:  services.ErrSettingsNewPasswordMustDiffer,
			want: settingsFormErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "new password must differ"),
		},
		{
			name: "weak password",
			err:  services.ErrSettingsWeakPassword,
			want: settingsFormErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "weak password"),
		},
		{
			name: "hash failed",
			err:  services.ErrSettingsPasswordHashFailed,
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to secure password"),
		},
		{
			name: "recovery code failed",
			err:  services.ErrSettingsRecoveryCodeGenerateFailed,
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to secure password"),
		},
		{
			name: "update failed",
			err:  services.ErrSettingsPasswordUpdateFailed,
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to update password"),
		},
		{
			name: "unknown",
			err:  errors.New("unknown"),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to update password"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if got := mapSettingsPasswordChangeError(testCase.err); got != testCase.want {
				t.Fatalf("unexpected mapped error: got %#v want %#v", got, testCase.want)
			}
		})
	}
}

func TestMapRecoveryCodeRegenerationError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want APIErrorSpec
	}{
		{
			name: "generate failed",
			err:  services.ErrRecoveryCodeGenerate,
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to create recovery code"),
		},
		{
			name: "update failed",
			err:  services.ErrRecoveryCodeUpdate,
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to update recovery code"),
		},
		{
			name: "unknown",
			err:  errors.New("unknown"),
			want: globalErrorSpec(fiber.StatusInternalServerError, APIErrorCategoryInternal, "failed to update recovery code"),
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if got := mapRecoveryCodeRegenerationError(testCase.err); got != testCase.want {
				t.Fatalf("unexpected mapped error: got %#v want %#v", got, testCase.want)
			}
		})
	}
}

func TestAPIErrorSpecIsFormError(t *testing.T) {
	testCases := []struct {
		name string
		spec APIErrorSpec
		want bool
	}{
		{
			name: "global",
			spec: globalErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid input"),
			want: false,
		},
		{
			name: "auth form",
			spec: authFormErrorSpec(fiber.StatusUnauthorized, APIErrorCategoryUnauthorized, "invalid credentials"),
			want: true,
		},
		{
			name: "settings form",
			spec: settingsFormErrorSpec(fiber.StatusBadRequest, APIErrorCategoryValidation, "invalid settings input"),
			want: true,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			if got := testCase.spec.IsFormError(); got != testCase.want {
				t.Fatalf("IsFormError() = %t, want %t", got, testCase.want)
			}
		})
	}
}
