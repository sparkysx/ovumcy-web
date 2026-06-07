package services

import (
	"testing"
	"time"
)

// Targeted coverage for two security-sensitive boundary mutants that survived the
// broad coverage pass. These guard the recovery-code alphabet and the login
// lockout configuration, where the exact comparison matters.

func TestIsCanonicalRecoveryCodeBody_AlphabetBoundaries(t *testing.T) {
	// The exact edges of the accepted alphabet (A, Z, 0, 9) must be accepted.
	if !isCanonicalRecoveryCodeBody("AZ09AZ09AZ09") {
		t.Fatal("expected the alphabet edges A/Z/0/9 to be accepted")
	}

	// The character immediately outside each accepted range must be rejected.
	// This kills boundary mutations on `c >= 'A'`, `c <= 'Z'`, `c >= '0'`, `c <= '9'`.
	for _, body := range []string{
		"@Z09AZ09AZ09", // '@' is one below 'A'
		"A[09AZ09AZ09", // '[' is one above 'Z'
		"AZ/9AZ09AZ09", // '/' is one below '0'
		"AZ0:AZ09AZ09", // ':' is one above '9'
	} {
		if isCanonicalRecoveryCodeBody(body) {
			t.Fatalf("expected out-of-range character in %q to be rejected", body)
		}
	}

	// Length boundary: only exactly 12 characters is canonical.
	if isCanonicalRecoveryCodeBody("AZ09AZ09AZ0") {
		t.Fatal("expected an 11-character body to be rejected")
	}
	if isCanonicalRecoveryCodeBody("AZ09AZ09AZ090") {
		t.Fatal("expected a 13-character body to be rejected")
	}
}

func TestAuthAttemptPolicyConfigure_AttemptsLowerBound(t *testing.T) {
	// Start with a known non-default attempts value, then reconfigure to 1.
	policy := NewAuthAttemptPolicy("login", nil, 8, time.Minute)

	// attempts == 1 is valid (lock after a single failure) and must be applied.
	// This kills the `attempts >= 1` -> `attempts > 1` boundary mutation.
	policy.Configure(1, time.Minute)
	if policy.attempts != 1 {
		t.Fatalf("expected Configure(1) to set attempts to 1, got %d", policy.attempts)
	}

	// Zero and negative attempts are rejected; the previous value is kept.
	policy.Configure(0, time.Minute)
	if policy.attempts != 1 {
		t.Fatalf("expected Configure(0) to be ignored, attempts still 1, got %d", policy.attempts)
	}
}
