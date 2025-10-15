package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isHeaderSeparatorRow(cells []string) bool {
	// Check if this is a header row with multiple city/region names in one cell
	for _, cell := range cells {
		cellLower := strings.ToLower(cell)
		// Count how many city/region names appear in a single cell
		cityCount := 0
		cities := []string{"augsburg", "bamberg", "freising", "münchen", "nürnberg", "regensburg", "rosenheim", "traunstein"}
		regions := []string{"oberbayern", "oberpfalz", "oberfranken", "mittelfranken", "schwaben"}

		for _, city := range cities {
			if strings.Contains(cellLower, city) {
				cityCount++
			}
		}
		for _, region := range regions {
			if strings.Contains(cellLower, region) {
				cityCount++
			}
		}

		// If a single cell contains multiple cities/regions, it's a header row
		if cityCount > 2 {
			return true
		}

		// Also check for "frei belegt" pattern
		if strings.Contains(cellLower, "frei") && strings.Contains(cellLower, "belegt") {
			return true
		}
	}
	return false
}

func isValidExamRow(cells []string) bool {
	// Check if this looks like a valid exam appointment row
	if len(cells) < 3 {
		return false
	}
	// First cell should look like a date/time (contains date pattern and time)
	firstCell := strings.TrimSpace(cells[0])
	// Look for date pattern (DD.MM.YYYY) and time pattern (HH:MM)
	hasDatePattern := strings.Contains(firstCell, ".") && strings.Contains(firstCell, ",")
	hasTimePattern := strings.Contains(firstCell, ":")
	return hasDatePattern && hasTimePattern
}

func parseDateTime(dateTimeStr string) time.Time {
	// Parse German date format: "25.10.2025, 08:00"
	layout := "02.01.2006, 15:04"
	parsed, err := time.Parse(layout, dateTimeStr)
	if err != nil {
		// Return zero time if parsing fails, will sort to beginning
		return time.Time{}
	}
	return parsed
}

func sortExamData(examData []map[string]string) {
	sort.Slice(examData, func(i, j int) bool {
		// Primary sort: by date_time
		dateI := parseDateTime(examData[i]["date_time"])
		dateJ := parseDateTime(examData[j]["date_time"])

		if !dateI.Equal(dateJ) {
			return dateI.Before(dateJ)
		}

		// Secondary sort: by location (alphabetical)
		locationI := examData[i]["location"]
		locationJ := examData[j]["location"]

		if locationI != locationJ {
			return locationI < locationJ
		}

		// Tertiary sort: by status (Frei before Belegt for same location/time)
		statusI := examData[i]["status"]
		statusJ := examData[j]["status"]

		return statusI < statusJ
	})
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


	if resp.StatusCode != 200 {
		log.Fatal("Error: Status code", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Error parsing HTML:", err)
	}


	var tables []map[string]interface{}

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		tableText := strings.TrimSpace(table.Text())

		// Capture any table that looks like it contains exam-related data
		if len(tableText) > 50 { // Skip very small tables
			var rows [][]string

			table.Find("tr").Each(func(j int, row *goquery.Selection) {
				var cells []string
				row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
					cellText := strings.TrimSpace(cell.Text())
					// Clean up text by removing extra whitespace and tabs
					cellText = strings.ReplaceAll(cellText, "\t", " ")
					cellText = strings.ReplaceAll(cellText, "\n", " ")
					// Normalize multiple spaces to single space
					for strings.Contains(cellText, "  ") {
						cellText = strings.ReplaceAll(cellText, "  ", " ")
					}
					cellText = strings.TrimSpace(cellText)

					// Skip empty cells and header separators
					if cellText != "" && cellText != "-" && !strings.Contains(cellText, "Augsburg\tBamberg") {
						cells = append(cells, cellText)
					}
				})
				// Only include rows that are valid exam appointments
				if len(cells) > 2 && !isHeaderSeparatorRow(cells) && isValidExamRow(cells) {
					rows = append(rows, cells)
				}
			})

			if len(rows) > 0 {
				// Convert rows to structured exam data
				var examData []map[string]string
				for _, row := range rows {
					if len(row) >= 4 { // Ensure we have enough columns
						exam := map[string]string{
							"date_time": row[0],
							"location":  row[1],
							"city":      row[2],
							"region":    row[3],
						}
						if len(row) >= 5 {
							exam["status"] = row[4]
						}
						examData = append(examData, exam)
					}
				}

				if len(examData) > 0 {
					// Sort the exam data for consistent ordering
					sortExamData(examData)

					tableData := map[string]interface{}{
						"exam_appointments": examData,
						"total_count":       len(examData),
					}
					tables = append(tables, tableData)
				}
			}
		}
	})

	jsonOutput, err := json.MarshalIndent(tables, "", "  ")
	if err != nil {
		log.Fatal("Error creating JSON:", err)
	}

	fmt.Print(string(jsonOutput))
}