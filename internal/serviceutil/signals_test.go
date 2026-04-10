package serviceutil

import "testing"

func TestContainsBounceSignature(t *testing.T) {
	if !ContainsBounceSignature("delivery status notification (failure)") {
		t.Fatal("expected bounce signature to be detected")
	}
	if ContainsBounceSignature("hello there, this is not a DSN") {
		t.Fatal("expected non-bounce text to be ignored")
	}
}

func TestSummarizeLocalSignals(t *testing.T) {
	got := SummarizeLocalSignals("support@gmail.com", "Base signal.")
	want := "Base signal. Consumer mailbox domain. Role-based local part."
	if got != want {
		t.Fatalf("SummarizeLocalSignals() = %q, want %q", got, want)
	}
}

func TestBounceReasonCodeAndSummary(t *testing.T) {
	if code := BounceReasonCode("token_match"); code != "bounce_token_match" {
		t.Fatalf("BounceReasonCode(token_match) = %q", code)
	}
	if code := BounceReasonCode("recipient_match"); code != "bounce_recipient_match" {
		t.Fatalf("BounceReasonCode(recipient_match) = %q", code)
	}
	if code := BounceReasonCode("anything_else"); code != "bounce_generic_dsn" {
		t.Fatalf("BounceReasonCode(default) = %q", code)
	}

	if summary := BounceSignalSummary("token_match"); summary != "Bounce evidence matched the unique probe token." {
		t.Fatalf("BounceSignalSummary(token_match) = %q", summary)
	}
}

func TestProbePathForReason(t *testing.T) {
	if got := ProbePathForReason("direct_path_unavailable"); got != "probe_bounce" {
		t.Fatalf("ProbePathForReason(direct_path_unavailable) = %q", got)
	}
	if got := ProbePathForReason("provider_policy_block"); got != "hybrid" {
		t.Fatalf("ProbePathForReason(provider_policy_block) = %q", got)
	}
}
