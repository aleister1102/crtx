package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

func handleSimpleSearch(domains []string, orgName string, concurrency int) {
	queriesChan := make(chan string, concurrency)
	resultsChan := make(chan string)
	var wg sync.WaitGroup
	client := createHTTPClient()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for query := range queriesChan {
				processQuery(query, client, resultsChan)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	go func() {
		defer close(queriesChan)
		if orgName != "" {
			queriesChan <- orgName
		} else {
			for _, domain := range domains {
				queriesChan <- "%." + domain
			}
		}
	}()

	printUniqueResults(resultsChan)
}

func handleRecursiveSearch(initialDomains []string, concurrency int) {
	client := createHTTPClient()

	allFoundSubdomains := NewSet()
	allFoundOrgs := NewSet()

	// Stage 1: Initial domain search
	fmt.Fprintln(os.Stderr, "[+] Stage 1: Finding initial subdomains and organizations...")
	var wg sync.WaitGroup
	queryChan := make(chan string, concurrency*2)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
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
		}()
	}

	for _, domain := range initialDomains {
		queryChan <- domain
		queryChan <- "%." + domain
	}
	close(queryChan)
	wg.Wait()

	stage1Subdomains := allFoundSubdomains.Copy()
	stage1Orgs := allFoundOrgs.Copy()
	fmt.Fprintf(os.Stderr, "[+] Stage 1: Found %d unique subdomains and %d unique organizations.\n", stage1Subdomains.Length(), stage1Orgs.Length())

	// Stage 2: Pivot on Organization Names
	fmt.Fprintln(os.Stderr, "[+] Stage 2: Searching based on discovered organizations...")
	orgQueryChan := make(chan string, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
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
		}()
	}

	for org := range stage1Orgs.items {
		orgQueryChan <- org
	}
	close(orgQueryChan)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "[+] Stage 2: Total unique domains after org search: %d.\n", allFoundSubdomains.Length())

	// Stage 3: Pivot on Subdomains
	fmt.Fprintln(os.Stderr, "[+] Stage 3: Searching based on discovered subdomains...")
	subdomainQueryChan := make(chan string, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
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
		}()
	}
	for subdomain := range stage1Subdomains.items {
		subdomainQueryChan <- subdomain
	}
	close(subdomainQueryChan)
	wg.Wait()
	fmt.Fprintf(os.Stderr, "[+] Stage 3: Total unique domains after subdomain pivot: %d.\n", allFoundSubdomains.Length())

	// Stage 4: Filter and print
	fmt.Fprintln(os.Stderr, "[+] Stage 4: Filtering and printing results...")
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
