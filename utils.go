package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// stringSlice is a custom flag type to handle multiple domain flags.
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// blockedSuffixes contains common domain suffixes to filter out.
var blockedSuffixes = []string{
	"cloudflaressl.com",
	"cloudflare.com",
	"pki.goog",
	"sectigo.com",
	"digicert.com",
	"comodoca.com",
	"usertrust.com",
	"godaddy.com",
	"hydrantid.com",
	"globalsign.com",
}

// isInputFromPipe checks if the program is receiving input from a pipe.
func isInputFromPipe() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// isDomainBlocked checks if a domain should be filtered based on blockedSuffixes.
func isDomainBlocked(domain string) bool {
	for _, suffix := range blockedSuffixes {
		if strings.HasSuffix(domain, "."+suffix) || domain == suffix {
			return true
		}
	}
	return false
}

// loadAdditionalBlockedSuffixes loads more suffixes to block from a given file path.
func loadAdditionalBlockedSuffixes(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not open blocklist file '%s': %v\n", filePath, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		suffix := strings.TrimSpace(scanner.Text())
		if suffix != "" {
			blockedSuffixes = append(blockedSuffixes, suffix)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error reading blocklist file '%s': %v\n", filePath, err)
	}
}

// printUniqueResults prints unique strings from a channel to stdout.
func printUniqueResults(resultsChan <-chan string) {
	unique := make(map[string]struct{})
	for res := range resultsChan {
		if _, exists := unique[res]; !exists {
			unique[res] = struct{}{}
			fmt.Println(res)
		}
	}
}
