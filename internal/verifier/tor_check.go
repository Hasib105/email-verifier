package verifier

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type TorCheckResult struct {
	IsTor   bool   `json:"is_tor"`
	IP      string `json:"ip"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// CheckTor verifies that outbound connections go through the Tor network.
// It connects to check.torproject.org via the configured SOCKS5 proxy.
func (v *EmailVerifier) CheckTor() TorCheckResult {
	if v.ProxyAddr == "" {
		return TorCheckResult{
			IsTor:   false,
			Message: "no SOCKS proxy configured",
		}
	}

	proxyURL, err := url.Parse("socks5://" + v.ProxyAddr)
	if err != nil {
		return TorCheckResult{Error: fmt.Sprintf("bad proxy URL: %v", err)}
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get("https://check.torproject.org/api/ip")
	if err != nil {
		return TorCheckResult{
			IsTor:   false,
			Error:   fmt.Sprintf("Tor connectivity check failed: %v", err),
			Message: "cannot reach Tor check service — is Tor running?",
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TorCheckResult{Error: fmt.Sprintf("read error: %v", err)}
	}

	var torResp struct {
		IsTor bool   `json:"IsTor"`
		IP    string `json:"IP"`
	}
	if err := json.Unmarshal(body, &torResp); err != nil {
		return TorCheckResult{
			Error:   fmt.Sprintf("parse error: %v", err),
			Message: string(body),
		}
	}

	result := TorCheckResult{
		IsTor: torResp.IsTor,
		IP:    torResp.IP,
	}

	if torResp.IsTor {
		result.Message = "Traffic is routed through Tor network"
	} else {
		result.Message = "WARNING: Traffic is NOT going through Tor"
	}

	return result
}
