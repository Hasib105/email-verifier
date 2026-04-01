package service

import "testing"

func TestContainsBounceSignature(t *testing.T) {
	if !containsBounceSignature("delivery status notification (failure)") {
		t.Fatal("expected bounce signature to be detected")
	}
	if containsBounceSignature("hello there, this is not a DSN") {
		t.Fatal("expected non-bounce text to be ignored")
	}
}

func TestSummarizeLocalSignals(t *testing.T) {
	got := summarizeLocalSignals("support@gmail.com", "Base signal.")
	want := "Base signal. Consumer mailbox domain. Role-based local part."
	if got != want {
		t.Fatalf("summarizeLocalSignals() = %q, want %q", got, want)
	}
}

func TestBounceReasonCodeAndSummary(t *testing.T) {
	if code := bounceReasonCode("token_match"); code != "bounce_token_match" {
		t.Fatalf("bounceReasonCode(token_match) = %q", code)
	}
	if code := bounceReasonCode("recipient_match"); code != "bounce_recipient_match" {
		t.Fatalf("bounceReasonCode(recipient_match) = %q", code)
	}
	if code := bounceReasonCode("anything_else"); code != "bounce_generic_dsn" {
		t.Fatalf("bounceReasonCode(default) = %q", code)
	}

	if summary := bounceSignalSummary("token_match"); summary != "Bounce evidence matched the unique probe token." {
		t.Fatalf("bounceSignalSummary(token_match) = %q", summary)
	}
}

func TestProbePathForReason(t *testing.T) {
	if got := probePathForReason("direct_path_unavailable"); got != "probe_bounce" {
		t.Fatalf("probePathForReason(direct_path_unavailable) = %q", got)
	}
	if got := probePathForReason("provider_policy_block"); got != "hybrid" {
		t.Fatalf("probePathForReason(provider_policy_block) = %q", got)
	}
}
