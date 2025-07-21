package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

func handleSimpleSearch(domains []string, orgName string, concurrency int) {
	logVerbose("Setting up worker pool with %d goroutines", concurrency)
	queriesChan := make(chan string, concurrency)
	resultsChan := make(chan string)
	var wg sync.WaitGroup
	client := createHTTPClient()

	// Start worker goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for query := range queriesChan {
				processQuery(query, client, resultsChan)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	go func() {
		defer close(queriesChan)
		if orgName != "" {
			logVerbose("Querying organization: %s", orgName)
			queriesChan <- orgName
		} else {
			logVerbose("Querying %d domains", len(domains))
			for _, domain := range domains {
				query := "%." + domain
				queriesChan <- query
			}
		}
		logVerbose("All queries queued")
	}()

	logVerbose("Starting to collect results...")
	printUniqueResults(resultsChan)
	logVerbose("Search completed")
}

func handleRecursiveSearch(initialDomains []string, concurrency int) {
	logVerbose("Starting recursive search with %d initial domains", len(initialDomains))
	client := createHTTPClient()

	allFoundSubdomains := NewSet()
	allFoundOrgs := NewSet()

	// Stage 1: Initial domain search
	fmt.Fprintln(os.Stderr, "[+] Stage 1: Finding initial subdomains and organizations...")
	logVerbose("Stage 1: Processing %d initial domains with %d workers", len(initialDomains), concurrency)
	var wg sync.WaitGroup
	queryChan := make(chan string, concurrency*2)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for query := range queryChan {
				entries, err := fetchCertsForQuery(query, client)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					extractDataFromEntry(entry, allFoundSubdomains, allFoundOrgs)
				}
			}
		}(i)
	}

	for _, domain := range initialDomains {
		queryChan <- domain
		queryChan <- "%." + domain
	}
	close(queryChan)
	wg.Wait()

	stage1Subdomains := allFoundSubdomains.Copy()
	stage1Orgs := allFoundOrgs.Copy()
	logVerbose("Stage 1 completed: %d subdomains, %d organizations", stage1Subdomains.Length(), stage1Orgs.Length())
	fmt.Fprintf(os.Stderr, "[+] Stage 1: Found %d unique subdomains and %d unique organizations.\n", stage1Subdomains.Length(), stage1Orgs.Length())

	// Stage 2: Pivot on Organization Names
	fmt.Fprintln(os.Stderr, "[+] Stage 2: Searching based on discovered organizations...")
	logVerbose("Stage 2: Processing %d organizations with %d workers", stage1Orgs.Length(), concurrency)
	orgQueryChan := make(chan string, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for org := range orgQueryChan {
				entries, err := fetchCertsForQuery(org, client)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					extractDataFromEntry(entry, allFoundSubdomains, nil)
				}
			}
		}(i)
	}

	logVerbose("Stage 2: Processing %d organizations", stage1Orgs.Length())
	for org := range stage1Orgs.items {
		orgQueryChan <- org
	}
	close(orgQueryChan)
	wg.Wait()
	logVerbose("Stage 2 completed: %d total domains", allFoundSubdomains.Length())
	fmt.Fprintf(os.Stderr, "[+] Stage 2: Total unique domains after org search: %d.\n", allFoundSubdomains.Length())

	// Stage 3: Pivot on Subdomains
	fmt.Fprintln(os.Stderr, "[+] Stage 3: Searching based on discovered subdomains...")
	logVerbose("Stage 3: Processing %d subdomains with %d workers", stage1Subdomains.Length(), concurrency)
	subdomainQueryChan := make(chan string, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for subdomain := range subdomainQueryChan {
				entries, err := fetchCertsForQuery(subdomain, client)
				if err != nil {
					continue
				}
				for _, entry := range entries {
					extractDataFromEntry(entry, allFoundSubdomains, nil)
				}
			}
		}(i)
	}

	logVerbose("Stage 3: Processing %d subdomains", stage1Subdomains.Length())
	for subdomain := range stage1Subdomains.items {
		logVerbose("Stage 3: Queuing subdomain: %s", subdomain)
		subdomainQueryChan <- subdomain
	}
	close(subdomainQueryChan)
	wg.Wait()
	logVerbose("Stage 3 completed: %d total domains", allFoundSubdomains.Length())
	fmt.Fprintf(os.Stderr, "[+] Stage 3: Total unique domains after subdomain pivot: %d.\n", allFoundSubdomains.Length())

	// Stage 4: Filter and print
	fmt.Fprintln(os.Stderr, "[+] Stage 4: Filtering and printing results...")
	logVerbose("Stage 4: Filtering %d domains for %d initial domains", allFoundSubdomains.Length(), len(initialDomains))
	finalResults := NewSet()
	for subdomain := range allFoundSubdomains.items {
		isSub := false
		for _, initialDomain := range initialDomains {
			if strings.HasSuffix(subdomain, "."+initialDomain) || subdomain == initialDomain {
				isSub = true
				break
			}
		}
		if isSub {
			finalResults.Add(subdomain)
		}
	}

	logVerbose("Stage 4: Filtered to %d final results", finalResults.Length())
	for _, res := range finalResults.Sorted() {
		fmt.Println(res)
	}
}

func processQuery(query string, client *http.Client, results chan<- string) {
	entries, err := fetchCertsForQuery(query, client)
	if err != nil {
		return
	}
	for _, entry := range entries {
		extractAndSend(entry, results)
	}
}
