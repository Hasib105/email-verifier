package verifier

import (
	"net/textproto"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEmail  string
		wantDomain string
		wantErr    bool
	}{
		{name: "normalizes case and whitespace", input: "  User@Example.com ", wantEmail: "user@example.com", wantDomain: "example.com"},
		{name: "rejects display name syntax", input: "User <user@example.com>", wantErr: true},
		{name: "rejects empty", input: "", wantErr: true},
		{name: "rejects repeated local dot", input: "user..name@example.com", wantErr: true},
		{name: "rejects bad domain spacing", input: "user@exa mple.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEmail, gotDomain, err := normalizeEmail(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeEmail() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if gotEmail != tt.wantEmail {
				t.Fatalf("normalizeEmail() email = %q, want %q", gotEmail, tt.wantEmail)
			}
			if gotDomain != tt.wantDomain {
				t.Fatalf("normalizeEmail() domain = %q, want %q", gotDomain, tt.wantDomain)
			}
		})
	}
}

func TestClassifySMTPErr(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantOutcome string
		wantCode    int
	}{
		{
			name:        "hard reject stays rejected",
			err:         &textproto.Error{Code: 550, Msg: "5.1.1 mailbox unavailable"},
			wantOutcome: "rejected",
			wantCode:    550,
		},
		{
			name:        "policy reject maps to policy",
			err:         &textproto.Error{Code: 550, Msg: "5.7.1 access denied by policy"},
			wantOutcome: "policy",
			wantCode:    550,
		},
		{
			name:        "tempfail maps to tempfail",
			err:         &textproto.Error{Code: 451, Msg: "4.7.1 please try again later"},
			wantOutcome: "tempfail",
			wantCode:    451,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outcome, code, _ := classifySMTPErr(tt.err, "fallback")
			if outcome != tt.wantOutcome {
				t.Fatalf("classifySMTPErr() outcome = %q, want %q", outcome, tt.wantOutcome)
			}
			if code != tt.wantCode {
				t.Fatalf("classifySMTPErr() code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}

func TestDirectSignalSummary(t *testing.T) {
	summary := directSignalSummary("base summary", true, true)
	if summary != "base summary Provider is on the local strict-provider list. Domain used A/AAAA fallback because no MX records were present." {
		t.Fatalf("unexpected summary: %q", summary)
	}
}
