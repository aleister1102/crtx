package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "crtx - A powerful subdomain enumeration tool using crt.sh\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  crtx [options]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  cat domains.txt | crtx\n")
		fmt.Fprintf(os.Stderr, "  crtx -d example.com\n")
		fmt.Fprintf(os.Stderr, "  crtx -d example.com -d anotherexample.com\n")
		fmt.Fprintf(os.Stderr, "  crtx -o \"Example Inc\"\n")
		fmt.Fprintf(os.Stderr, "  crtx -r -d example.com\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
}

func main() {
	var domains stringSlice
	flag.Var(&domains, "d", "Domain to search for (can be specified multiple times)")
	concurrency := flag.Int("c", 50, "Set the concurrency level")
	orgName := flag.String("o", "", "Organization name to search for")
	recursive := flag.Bool("r", false, "Perform a recursive search (requires -d)")
	blocklistFile := flag.String("bf", "", "Path to a file containing additional domain suffixes to block")
	flag.Parse()

	if *blocklistFile != "" {
		loadAdditionalBlockedSuffixes(*blocklistFile)
	}

	allInputDomains := gatherInputDomains(domains)
	hasDomains := len(allInputDomains) > 0
	hasOrg := *orgName != ""

	if err := validateArgs(hasDomains, hasOrg, *recursive); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if !hasDomains && !hasOrg {
		flag.Usage()
		os.Exit(1)
	}

	if *recursive {
		handleRecursiveSearch(allInputDomains, *concurrency)
	} else {
		handleSimpleSearch(allInputDomains, *orgName, *concurrency)
	}
}

// gatherInputDomains collects domains from command-line flags and standard input.
func gatherInputDomains(initialDomains []string) []string {
	allInputDomains := initialDomains
	if isInputFromPipe() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			domain := strings.TrimSpace(scanner.Text())
			if domain != "" {
				allInputDomains = append(allInputDomains, domain)
			}
		}
	}
	return allInputDomains
}

// validateArgs checks for unsupported combinations of command-line arguments.
func validateArgs(hasDomains, hasOrg, isRecursive bool) error {
	if hasOrg && hasDomains {
		return fmt.Errorf("cannot use domain flags (-d) or stdin and organization flag (-o) together")
	}
	if isRecursive && hasOrg {
		return fmt.Errorf("organization search (-o) cannot be combined with recursive search (-r)")
	}
	if isRecursive && !hasDomains {
		return fmt.Errorf("recursive search (-r) requires input from -d flags or stdin")
	}
	return nil
}
