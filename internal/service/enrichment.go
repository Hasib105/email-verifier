package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"email-verifier-api/internal/store"

	"github.com/google/uuid"
)

type EnrichmentResult struct {
	ConfidenceScore int
	RiskLevel       string
	Summary         string
	Evidence        []store.EnrichmentEvidence
}

type ExternalSignalProvider interface {
	Name() string
	Enabled() bool
	Lookup(ctx context.Context, email string, domain string) ([]store.EnrichmentEvidence, error)
}

type DisabledProvider struct {
	name string
}

func (p DisabledProvider) Name() string  { return p.name }
func (p DisabledProvider) Enabled() bool { return false }
func (p DisabledProvider) Lookup(context.Context, string, string) ([]store.EnrichmentEvidence, error) {
	return nil, nil
}

type EnrichmentService struct {
	httpClient *http.Client
	providers  []ExternalSignalProvider
}

func NewEnrichmentService() *EnrichmentService {
	return &EnrichmentService{
		httpClient: &http.Client{Timeout: 8 * time.Second},
		providers: []ExternalSignalProvider{
			DisabledProvider{name: "hunter"},
			DisabledProvider{name: "abstractapi"},
		},
	}
}

func (s *EnrichmentService) Enrich(ctx context.Context, record *store.VerificationRecord) (EnrichmentResult, error) {
	score := record.ConfidenceScore
	evidence := []store.EnrichmentEvidence{}
	addEvidence := func(source, kind, signal string, weight int, summary string) {
		evidence = append(evidence, store.EnrichmentEvidence{
			ID:             uuid.NewString(),
			VerificationID: record.ID,
			Source:         source,
			Kind:           kind,
			Signal:         signal,
			Weight:         weight,
			Summary:        summary,
			CreatedAt:      time.Now().Unix(),
		})
		score += weight
	}

	localPart, _, _ := strings.Cut(record.Email, "@")
	if isDisposableDomain(record.Domain) {
		addEvidence("first_party", "domain", "disposable_domain", -50, fmt.Sprintf("%s is a known disposable domain", record.Domain))
	}
	if isFreeMailboxDomain(record.Domain) {
		addEvidence("first_party", "domain", "consumer_mailbox_provider", -10, fmt.Sprintf("%s is a consumer mailbox provider", record.Domain))
	}
	if isRoleMailbox(localPart) {
		addEvidence("first_party", "mailbox", "role_based_local_part", -10, fmt.Sprintf("%s looks like a role-based mailbox", localPart))
	}

	publicEmails, pageSignals, err := s.inspectDomainWebsite(ctx, record.Domain)
	if err == nil {
		for _, pageSignal := range pageSignals {
			addEvidence("first_party", "web", pageSignal.signal, pageSignal.weight, pageSignal.summary)
		}

		exactMatch := false
		for _, publicEmail := range publicEmails {
			if strings.EqualFold(publicEmail, record.Email) {
				exactMatch = true
				break
			}
		}
		if exactMatch {
			addEvidence("first_party", "web", "exact_public_email_match", 35, fmt.Sprintf("%s appears publicly on the company website", record.Email))
		} else if len(publicEmails) > 0 {
			addEvidence("first_party", "web", "same_domain_public_addresses", 15, fmt.Sprintf("found %d public same-domain addresses on the company website", len(publicEmails)))
			if looksPatternBased(localPart) {
				addEvidence("first_party", "heuristic", "matches_common_business_pattern", 10, fmt.Sprintf("%s follows a common business-email naming pattern", localPart))
			}
		}
	}

	for _, provider := range s.providers {
		if !provider.Enabled() {
			continue
		}
		items, err := provider.Lookup(ctx, record.Email, record.Domain)
		if err != nil {
			addEvidence(provider.Name(), "provider", "provider_lookup_failed", -5, fmt.Sprintf("%s lookup failed: %v", provider.Name(), err))
			continue
		}
		for _, item := range items {
			evidence = append(evidence, item)
			score += item.Weight
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return EnrichmentResult{
		ConfidenceScore: score,
		RiskLevel:       deriveRiskLevel(record.Classification, score),
		Summary:         summarizeEvidence(record.Classification, evidence),
		Evidence:        evidence,
	}, nil
}

type pageSignal struct {
	signal  string
	weight  int
	summary string
}

var emailPattern = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)

func (s *EnrichmentService) inspectDomainWebsite(ctx context.Context, domain string) ([]string, []pageSignal, error) {
	urls := []string{"https://" + domain, "https://" + domain + "/contact", "http://" + domain}
	seenEmails := map[string]struct{}{}
	signals := []pageSignal{}

	for _, rawURL := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			continue
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			continue
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if readErr != nil {
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			signals = append(signals, pageSignal{
				signal:  "website_reachable",
				weight:  8,
				summary: fmt.Sprintf("%s responded with %d", rawURL, resp.StatusCode),
			})
		}

		for _, match := range emailPattern.FindAllString(strings.ToLower(string(body)), -1) {
			if strings.HasSuffix(match, "@"+strings.ToLower(domain)) {
				seenEmails[match] = struct{}{}
			}
		}
	}

	emails := make([]string, 0, len(seenEmails))
	for email := range seenEmails {
		emails = append(emails, email)
	}

	return emails, signals, nil
}

func deriveRiskLevel(classification string, score int) string {
	switch classification {
	case "deliverable":
		if score >= 80 {
			return "low"
		}
		return "medium"
	case "undeliverable":
		return "high"
	case "accept_all":
		if score >= 70 {
			return "medium"
		}
		return "high"
	default:
		if score >= 75 {
			return "medium"
		}
		return "high"
	}
}

func summarizeEvidence(classification string, evidence []store.EnrichmentEvidence) string {
	if len(evidence) == 0 {
		return fmt.Sprintf("No enrichment evidence raised confidence for %s", classification)
	}

	best := evidence[0]
	for _, item := range evidence[1:] {
		if item.Weight > best.Weight {
			best = item
		}
	}
	return best.Summary
}

func isDisposableDomain(domain string) bool {
	disposables := map[string]bool{
		"mailinator.com":    true,
		"tempmail.com":      true,
		"10minutemail.com":  true,
		"guerrillamail.com": true,
		"throwaway.email":   true,
	}
	return disposables[strings.ToLower(domain)]
}

func isFreeMailboxDomain(domain string) bool {
	free := map[string]bool{
		"gmail.com":      true,
		"googlemail.com": true,
		"yahoo.com":      true,
		"hotmail.com":    true,
		"outlook.com":    true,
		"icloud.com":     true,
	}
	return free[strings.ToLower(domain)]
}

func isRoleMailbox(localPart string) bool {
	switch strings.ToLower(localPart) {
	case "admin", "billing", "contact", "hello", "info", "legal", "sales", "support", "team":
		return true
	default:
		return false
	}
}

func looksPatternBased(localPart string) bool {
	if strings.Contains(localPart, ".") || strings.Contains(localPart, "_") {
		return true
	}
	if len(localPart) >= 6 && regexp.MustCompile(`^[a-z]+[0-9]*$`).MatchString(localPart) {
		return true
	}
	return false
}
