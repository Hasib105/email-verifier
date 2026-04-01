package service

import "testing"

func TestValidateServerHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{name: "valid fqdn", host: "smtp.gmail.com", wantErr: false},
		{name: "valid localhost", host: "localhost", wantErr: false},
		{name: "valid ip", host: "127.0.0.1", wantErr: false},
		{name: "invalid email-like host", host: "smtp@gmail.com", wantErr: true},
		{name: "invalid url-like host", host: "https://smtp.gmail.com", wantErr: true},
		{name: "invalid spaced host", host: "smtp gmail.com", wantErr: true},
		{name: "empty host", host: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerHost("host", tt.host)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateServerHost() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{name: "valid lower", port: 1, wantErr: false},
		{name: "valid smtp", port: 587, wantErr: false},
		{name: "valid upper", port: 65535, wantErr: false},
		{name: "invalid zero", port: 0, wantErr: true},
		{name: "invalid negative", port: -1, wantErr: true},
		{name: "invalid too high", port: 65536, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePort("port", tt.port)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePort() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeServerHost(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim and lowercase", in: " SMTP.GMAIL.COM ", want: "smtp.gmail.com"},
		{name: "normalize smtp at typo", in: "smtp@gmail.com", want: "smtp.gmail.com"},
		{name: "normalize imap at typo", in: "imap@gmail.com", want: "imap.gmail.com"},
		{name: "keep normal host", in: "mail.example.com", want: "mail.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeServerHost(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeServerHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInferIMAPHost(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "smtp prefix converts", in: "smtp.gmail.com", want: "imap.gmail.com"},
		{name: "smtp at typo converts", in: "smtp@gmail.com", want: "imap.gmail.com"},
		{name: "non smtp host unchanged", in: "mail.example.com", want: "mail.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferIMAPHost(tt.in)
			if got != tt.want {
				t.Fatalf("inferIMAPHost() = %q, want %q", got, tt.want)
			}
		})
	}
}
