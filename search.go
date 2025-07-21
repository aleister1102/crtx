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
			logVerbose("Worker %d started", workerID)
			for query := range queriesChan {
				logVerbose("Worker %d processing query: %s", workerID, query)
				processQuery(query, client, resultsChan)
			}
			logVerbose("Worker %d finished", workerID)
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
				logVerbose("Adding query to queue: %s", query)
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
			logVerbose("Stage 1 Worker %d started", workerID)
			for query := range queryChan {
				logVerbose("Stage 1 Worker %d processing: %s", workerID, query)
				entries, err := fetchCertsForQuery(query, client)
				if err != nil {
					logVerbose("Stage 1 Worker %d error for %s: %v", workerID, query, err)
					continue
				}
				logVerbose("Stage 1 Worker %d found %d entries for %s", workerID, len(entries), query)
				for _, entry := range entries {
					extractDataFromEntry(entry, allFoundSubdomains, allFoundOrgs)
				}
			}
			logVerbose("Stage 1 Worker %d finished", workerID)
		}(i)
	}

	for _, domain := range initialDomains {
		logVerbose("Stage 1: Queuing domain searches for %s", domain)
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
			logVerbose("Stage 2 Worker %d started", workerID)
			for org := range orgQueryChan {
				logVerbose("Stage 2 Worker %d processing org: %s", workerID, org)
				entries, err := fetchCertsForQuery(org, client)
				if err != nil {
					logVerbose("Stage 2 Worker %d error for %s: %v", workerID, org, err)
					continue
				}
				logVerbose("Stage 2 Worker %d found %d entries for org %s", workerID, len(entries), org)
				for _, entry := range entries {
					extractDataFromEntry(entry, allFoundSubdomains, nil)
				}
			}
			logVerbose("Stage 2 Worker %d finished", workerID)
		}(i)
	}

	logVerbose("Stage 2: Queuing %d organizations", stage1Orgs.Length())
	for org := range stage1Orgs.items {
		logVerbose("Stage 2: Queuing org: %s", org)
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
			logVerbose("Stage 3 Worker %d started", workerID)
			for subdomain := range subdomainQueryChan {
				logVerbose("Stage 3 Worker %d processing subdomain: %s", workerID, subdomain)
				entries, err := fetchCertsForQuery(subdomain, client)
				if err != nil {
					logVerbose("Stage 3 Worker %d error for %s: %v", workerID, subdomain, err)
					continue
				}
				logVerbose("Stage 3 Worker %d found %d entries for subdomain %s", workerID, len(entries), subdomain)
				for _, entry := range entries {
					extractDataFromEntry(entry, allFoundSubdomains, nil)
				}
			}
			logVerbose("Stage 3 Worker %d finished", workerID)
		}(i)
	}

	logVerbose("Stage 3: Queuing %d subdomains", stage1Subdomains.Length())
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
	logVerbose("Fetching certificates for query: %s", query)
	entries, err := fetchCertsForQuery(query, client)
	if err != nil {
		logVerbose("Error fetching certificates for %s: %v", query, err)
		return
	}
	logVerbose("Found %d certificate entries for query: %s", len(entries), query)
	for _, entry := range entries {
		extractAndSend(entry, results)
	}
}
