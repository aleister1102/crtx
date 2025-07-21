package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Global verbose flag
var verbose bool

// logVerbose prints a message to stderr if verbose mode is enabled
func logVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

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
		fmt.Fprintf(os.Stderr, "  crtx -r -d example.com\n")
		fmt.Fprintf(os.Stderr, "  crtx -v -d example.com  # verbose output\n\n")
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
	verboseFlag := flag.Bool("v", false, "Enable verbose output")
	flag.Parse()

	// Set global verbose flag
	verbose = *verboseFlag

	if *blocklistFile != "" {
		logVerbose("Loading additional blocked suffixes from: %s", *blocklistFile)
		loadAdditionalBlockedSuffixes(*blocklistFile)
	}

	logVerbose("Gathering input domains...")
	allInputDomains := gatherInputDomains(domains)
	logVerbose("Found %d input domains", len(allInputDomains))

	hasDomains := len(allInputDomains) > 0
	hasOrg := *orgName != ""

	logVerbose("Validating arguments...")
	if err := validateArgs(hasDomains, hasOrg, *recursive); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if !hasDomains && !hasOrg {
		flag.Usage()
		os.Exit(1)
	}

	if *recursive {
		logVerbose("Starting recursive search with %d domains and concurrency %d", len(allInputDomains), *concurrency)
		handleRecursiveSearch(allInputDomains, *concurrency)
	} else {
		logVerbose("Starting simple search with concurrency %d", *concurrency)
		if hasOrg {
			logVerbose("Searching for organization: %s", *orgName)
		} else {
			logVerbose("Searching for domains: %v", allInputDomains)
		}
		handleSimpleSearch(allInputDomains, *orgName, *concurrency)
	}
}

// gatherInputDomains collects domains from command-line flags and standard input.
func gatherInputDomains(initialDomains []string) []string {
	allInputDomains := initialDomains
	if isInputFromPipe() {
		logVerbose("Reading domains from stdin...")
		scanner := bufio.NewScanner(os.Stdin)
		count := 0
		for scanner.Scan() {
			domain := strings.TrimSpace(scanner.Text())
			if domain != "" {
				allInputDomains = append(allInputDomains, domain)
				count++
			}
		}
		logVerbose("Read %d domains from stdin", count)
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
