package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// crtshEntry matches the JSON structure from crt.sh.
type crtshEntry struct {
	IssuerName string `json:"issuer_name"`
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"`
}

// orgRegex extracts the Organization Name from an issuer string.
var orgRegex = regexp.MustCompile(`O=([^,]+)`)

// fetchCertsForQuery fetches certificate transparency logs from crt.sh.
func fetchCertsForQuery(query string, client *http.Client) ([]crtshEntry, error) {
	if query == "" {
		return nil, nil
	}
	requestURL := fmt.Sprintf("https://crt.sh/?q=%s&output=json", url.QueryEscape(query))

	const maxRetries = 3
	const retryDelay = 10 * time.Second
	var entries []crtshEntry

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			logVerbose("Retry attempt %d/%d for query: %s", attempt, maxRetries, query)
		}

		req, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "crtx/1.9")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(body, &entries); err != nil {
				return nil, err
			}
			logVerbose("Found %d entries for query: %s", len(entries), query)
			return entries, nil
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt < maxRetries {
				logVerbose("Rate limited for query %s, retrying in %v... (Attempt %d/%d)", query, retryDelay, attempt+1, maxRetries)
				fmt.Fprintf(os.Stderr, "[!] Received 429 Too Many Requests. Retrying in %v... (Attempt %d/%d for query: %s)\n", retryDelay, attempt+1, maxRetries, query)
				time.Sleep(retryDelay)
				continue
			}
			logVerbose("Max retries reached for query %s after 429 status", query)
			return nil, fmt.Errorf("max retries reached for query '%s' after 429 status", query)
		}

		resp.Body.Close()
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	return nil, fmt.Errorf("unexpected error after retries for query '%s'", query)
}

// extractAndSend extracts subdomains from a crtshEntry and sends them to a channel.
func extractAndSend(entry crtshEntry, results chan<- string) {
	names := []string{entry.CommonName, entry.NameValue}
	for _, nameField := range names {
		subdomains := strings.Split(nameField, "\n")
		for _, subdomain := range subdomains {
			clean := strings.TrimSpace(subdomain)
			if clean != "" && !strings.Contains(clean, "*") && !isDomainBlocked(clean) {
				results <- clean
			}
		}
	}
}

// extractDataFromEntry extracts subdomains and organization names from a crtshEntry.
func extractDataFromEntry(entry crtshEntry, subdomains *Set, orgs *Set) {
	names := []string{entry.CommonName, entry.NameValue}
	for _, nameField := range names {
		for _, subdomain := range strings.Split(nameField, "\n") {
			clean := strings.TrimSpace(subdomain)
			if clean != "" && !strings.Contains(clean, "*") && !isDomainBlocked(clean) {
				subdomains.Add(clean)
			}
		}
	}

	if orgs != nil {
		matches := orgRegex.FindStringSubmatch(entry.IssuerName)
		if len(matches) > 1 {
			orgs.Add(matches[1])
		}
	}
}

// createHTTPClient creates a new HTTP client with a specific timeout and TLS config.
func createHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 45 * time.Second,
	}
}
