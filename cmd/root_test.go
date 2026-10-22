package cmd

import (
	"strings"
	"testing"
)

func TestValidateRateLimitConfig(t *testing.T) {
	orig := flagRateLimit
	defer func() { flagRateLimit = orig }()

	for _, v := range []int{0, -1, -120} {
		flagRateLimit = v
		err := validateRateLimitConfig()
		if err == nil {
			t.Errorf("flagRateLimit = %d: expected error", v)
		} else if !strings.Contains(err.Error(), "--rate-limit") {
			t.Errorf("flagRateLimit = %d: error should name --rate-limit: %v", v, err)
		}
	}
	flagRateLimit = 120
	if err := validateRateLimitConfig(); err != nil {
		t.Errorf("flagRateLimit = 120: unexpected error %v", err)
	}
}

// TestInvalidRateLimitFlagFailsFast proves the wiring end to end: an
// invalid --rate-limit makes command execution fail in PersistentPreRunE,
// before any subcommand (or network call) runs.
func TestInvalidRateLimitFlagFailsFast(t *testing.T) {
	rootCmd.SetOut(&strings.Builder{})
	rootCmd.SetErr(&strings.Builder{})
	rootCmd.SetArgs([]string{"--rate-limit", "0", "auth", "status"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected execution to fail for --rate-limit 0")
	}
	if !strings.Contains(err.Error(), "--rate-limit") {
		t.Errorf("error should name --rate-limit: %v", err)
	}
	// Leave the shared flag state sane for any later test.
	flagRateLimit = rateLimitDefault
}
