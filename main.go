package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	// Start with the main page to establish session
	baseURL := "https://fischerpruefung-online.bayern.de/fprApp/"
	url := "https://fischerpruefung-online.bayern.de/fprApp/verwaltung/Pruefungssuche?execution=e9s1"

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 20 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// First visit the base page
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		log.Fatal("Error creating base request:", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	_, err = client.Do(req)
	if err != nil {
		log.Printf("Warning: Could not access base page: %v", err)
	}

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error fetching URL:", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Final URL: %s\n", resp.Request.URL.String())

	if resp.StatusCode != 200 {
		log.Fatal("Error: Status code", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
	}

	// Debug: Print page title and check for tables
	title := doc.Find("title").Text()
	fmt.Printf("Page title: %s\n", strings.TrimSpace(title))

	tableCount := doc.Find("table").Length()
	fmt.Printf("Total tables found: %d\n", tableCount)

	// Debug: Print first few lines of body text
	bodyText := strings.TrimSpace(doc.Find("body").Text())
	if len(bodyText) > 500 {
		bodyText = bodyText[:500] + "..."
	}
	fmt.Printf("Body text preview: %s\n\n", bodyText)

	var tables []map[string]interface{}

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		tableText := strings.TrimSpace(table.Text())
		fmt.Printf("Table %d text preview: %s\n", i+1, tableText[:min(200, len(tableText))])

		// Capture any table that looks like it contains exam-related data
		if len(tableText) > 50 { // Skip very small tables
			var rows [][]string

			table.Find("tr").Each(func(j int, row *goquery.Selection) {
				var cells []string
				row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
					cells = append(cells, strings.TrimSpace(cell.Text()))
				})
				if len(cells) > 0 {
					rows = append(rows, cells)
				}
			})

			if len(rows) > 0 {
				tableData := map[string]interface{}{
					"table_index": i + 1,
					"rows":        rows,
				}
				tables = append(tables, tableData)
			}
		}
	})

	jsonOutput, err := json.MarshalIndent(tables, "", "  ")
	if err != nil {
		log.Fatal("Error creating JSON:", err)
	}

	fmt.Print(string(jsonOutput))
}