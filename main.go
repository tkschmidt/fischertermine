package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ExamAppointment represents a fishing exam appointment with all details
type ExamAppointment struct {
	DateTime string `json:"date_time"`
	Location string `json:"location"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Status   string `json:"status"`

	// Detail fields (fetched from detail page)
	ExamVenue            string `json:"exam_venue,omitempty"`
	Room                 string `json:"room,omitempty"`
	PostalCode           string `json:"postal_code,omitempty"`
	Street               string `json:"street,omitempty"`
	HouseNumber          string `json:"house_number,omitempty"`
	ExamDate             string `json:"exam_date,omitempty"`
	ExamStartTime        string `json:"exam_start_time,omitempty"`
	Headphones           string `json:"headphones,omitempty"`
	WheelchairAccessible string `json:"wheelchair_accessible,omitempty"`
	MinParticipants      string `json:"min_participants,omitempty"`
	MaxParticipants      string `json:"max_participants,omitempty"`
	CurrentParticipants  string `json:"current_participants,omitempty"`
	DetailStatus         string `json:"detail_status,omitempty"`
}

// OutputData represents the JSON output structure
type OutputData struct {
	ExamAppointments []ExamAppointment `json:"exam_appointments"`
	TotalCount       int               `json:"total_count"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isHeaderSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		cellLower := strings.ToLower(cell)
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

		if cityCount > 2 {
			return true
		}

		if strings.Contains(cellLower, "frei") && strings.Contains(cellLower, "belegt") {
			return true
		}
	}
	return false
}

func isValidExamRow(cells []string) bool {
	if len(cells) < 3 {
		return false
	}
	firstCell := strings.TrimSpace(cells[0])
	hasDatePattern := strings.Contains(firstCell, ".") && strings.Contains(firstCell, ",")
	hasTimePattern := strings.Contains(firstCell, ":")
	return hasDatePattern && hasTimePattern
}

func parseDateTime(dateTimeStr string) time.Time {
	layout := "02.01.2006, 15:04"
	parsed, err := time.Parse(layout, dateTimeStr)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func sortExamData(examData []ExamAppointment) {
	sort.Slice(examData, func(i, j int) bool {
		dateI := parseDateTime(examData[i].DateTime)
		dateJ := parseDateTime(examData[j].DateTime)

		if !dateI.Equal(dateJ) {
			return dateI.Before(dateJ)
		}

		if examData[i].Location != examData[j].Location {
			return examData[i].Location < examData[j].Location
		}

		return examData[i].Status < examData[j].Status
	})
}

// findButtonForExam finds the submit button name for a specific exam in the list page
func findButtonForExam(listDoc *goquery.Document, targetExam ExamAppointment) string {
	var buttonName string

	listDoc.Find("table tr").Each(func(i int, row *goquery.Selection) {
		if buttonName != "" {
			return // Already found
		}

		var cells []string
		row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
			cellText := strings.TrimSpace(cell.Text())
			cellText = strings.ReplaceAll(cellText, "\t", " ")
			cellText = strings.ReplaceAll(cellText, "\n", " ")
			for strings.Contains(cellText, "  ") {
				cellText = strings.ReplaceAll(cellText, "  ", " ")
			}
			cellText = strings.TrimSpace(cellText)

			if cellText != "" && cellText != "-" {
				cells = append(cells, cellText)
			}
		})

		// Check if this row matches our target exam
		if len(cells) >= 3 &&
			cells[0] == targetExam.DateTime &&
			cells[1] == targetExam.Location {

			// Found the matching row, extract the button
			row.Find("input[type=submit].select").Each(func(k int, btn *goquery.Selection) {
				if name, exists := btn.Attr("name"); exists && buttonName == "" {
					buttonName = name
				}
			})
		}
	})

	return buttonName
}

// fetchDetailPage submits the form for a specific exam and returns the detail page HTML
func fetchDetailPage(client *http.Client, listDoc *goquery.Document, formAction string, buttonName string) (*goquery.Document, *http.Response, error) {
	// Extract all form fields
	formData := url.Values{}

	listDoc.Find("input").Each(func(i int, s *goquery.Selection) {
		name, nameExists := s.Attr("name")
		inputType, _ := s.Attr("type")
		value, _ := s.Attr("value")

		if nameExists && name != "" {
			// Skip submit buttons and unchecked checkboxes
			if inputType != "submit" && inputType != "image" {
				if inputType == "checkbox" {
					_, checked := s.Attr("checked")
					if !checked {
						return
					}
				}
				formData.Set(name, value)
			}
		}
	})

	// Add the specific submit button
	formData.Set(buttonName, "")

	// Build submit URL
	submitURL := "https://fischerpruefung-online.bayern.de" + formAction

	req, _ := http.NewRequest("POST", submitURL, strings.NewReader(formData.Encode()))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	resp.Body.Close()
	return doc, resp, err
}

// parseDetailPage extracts detail information from the detail view and updates the exam struct
func parseDetailPage(doc *goquery.Document, exam *ExamAppointment) {
	// Track which fields we've already set
	detailsSet := make(map[string]bool)

	// Find all text and look for label-value pairs
	var currentLabel string
	doc.Find("*").Each(func(i int, elem *goquery.Selection) {
		text := strings.TrimSpace(elem.Text())

		// Skip empty or very large text blocks (likely containers)
		if text == "" || len(text) > 200 {
			return
		}

		// Check if this might be a label (German field names)
		labels := []string{"Prüfungslokal", "Raum", "PLZ", "Ort", "Straße", "Hausnummer",
			"Prüfungstermin", "Prüfungsbeginn", "Kopfhörer", "Behindertengerecht",
			"Min. Teilnehmer", "Max. Teilnehmer", "Aktuelle Teilnehmer", "Status"}

		for _, label := range labels {
			if text == label {
				currentLabel = label
				return
			}
		}

		// If we have a current label and this might be a value
		if currentLabel != "" && text != currentLabel && len(text) < 100 {
			// Check if this element only contains the value (not nested with label)
			if !strings.Contains(text, currentLabel) {
				// Set the appropriate field based on the label
				if !detailsSet[currentLabel] {
					setExamField(exam, currentLabel, text)
					detailsSet[currentLabel] = true
				}
				currentLabel = ""
			}
		}
	})
}

// setExamField sets the appropriate field on the ExamAppointment struct based on the German label
func setExamField(exam *ExamAppointment, label, value string) {
	switch label {
	case "Prüfungslokal":
		exam.ExamVenue = value
	case "Raum":
		exam.Room = value
	case "PLZ":
		exam.PostalCode = value
	case "Straße":
		exam.Street = value
	case "Hausnummer":
		exam.HouseNumber = value
	case "Prüfungstermin":
		exam.ExamDate = value
	case "Prüfungsbeginn":
		exam.ExamStartTime = value
	case "Kopfhörer":
		exam.Headphones = value
	case "Behindertengerecht":
		exam.WheelchairAccessible = value
	case "Min. Teilnehmer":
		exam.MinParticipants = value
	case "Max. Teilnehmer":
		exam.MaxParticipants = value
	case "Aktuelle Teilnehmer":
		exam.CurrentParticipants = value
	case "Status":
		exam.DetailStatus = value
	}
}

func main() {
	baseURL := "https://fischerpruefung-online.bayern.de/fprApp/"
	listURL := "https://fischerpruefung-online.bayern.de/fprApp/verwaltung/Pruefungssuche?execution=e9s1"

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

	// Visit base page
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		log.Fatal("Error creating base request:", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	_, err = client.Do(req)
	if err != nil {
		log.Printf("Warning: Could not access base page: %v", err)
	}

	// Get list page
	req, err = http.NewRequest("GET", listURL, nil)
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

	// Get form action
	var formAction string
	doc.Find("form#pruefungsterminSearch").Each(func(i int, form *goquery.Selection) {
		action, exists := form.Attr("action")
		if exists {
			formAction = action
		}
	})

	if formAction == "" {
		log.Fatal("No form action found")
	}

	// Collect exam data (we'll find button names later in fresh sessions)
	var examsToFetch []ExamAppointment

	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		tableText := strings.TrimSpace(table.Text())

		if len(tableText) > 50 {
			table.Find("tr").Each(func(j int, row *goquery.Selection) {
				var cells []string

				row.Find("td, th").Each(func(k int, cell *goquery.Selection) {
					cellText := strings.TrimSpace(cell.Text())
					cellText = strings.ReplaceAll(cellText, "\t", " ")
					cellText = strings.ReplaceAll(cellText, "\n", " ")
					for strings.Contains(cellText, "  ") {
						cellText = strings.ReplaceAll(cellText, "  ", " ")
					}
					cellText = strings.TrimSpace(cellText)

					if cellText != "" && cellText != "-" && !strings.Contains(cellText, "Augsburg\tBamberg") {
						cells = append(cells, cellText)
					}
				})

				if len(cells) > 2 && !isHeaderSeparatorRow(cells) && isValidExamRow(cells) {
					exam := ExamAppointment{
						DateTime: cells[0],
						Location: cells[1],
						City:     cells[2],
						Region:   cells[3],
					}
					if len(cells) >= 5 {
						exam.Status = cells[4]
					}

					examsToFetch = append(examsToFetch, exam)
				}
			})
		}
	})

	log.Printf("Found %d exams, fetching details concurrently...\n", len(examsToFetch))

	// Fetch details for each exam concurrently using a worker pool
	examData := make([]ExamAppointment, len(examsToFetch))
	maxWorkers := 10
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range examsToFetch {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Acquire semaphore (limit concurrent requests)
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			exam := examsToFetch[index]
			log.Printf("Fetching details for exam %d/%d: %s at %s\n", index+1, len(examsToFetch), exam.DateTime, exam.Location)

			// Create a NEW client with fresh session for this exam
			examJar, _ := cookiejar.New(nil)
			examClient := &http.Client{
				Jar: examJar,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) >= 20 {
						return fmt.Errorf("too many redirects")
					}
					return nil
				},
			}

			// Visit base page to establish session
			req, _ := http.NewRequest("GET", baseURL, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
			examClient.Do(req)

			// Fetch the list page
			req, _ = http.NewRequest("GET", listURL, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			resp, err := examClient.Do(req)
			if err != nil {
				log.Printf("Error fetching list page for exam %d: %v\n", index+1, err)
				mu.Lock()
				examData[index] = exam // Store basic info only
				mu.Unlock()
				return
			}

			listDoc, err := goquery.NewDocumentFromReader(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Printf("Error parsing list page for exam %d: %v\n", index+1, err)
				mu.Lock()
				examData[index] = exam // Store basic info only
				mu.Unlock()
				return
			}

			// Get form action from THIS fresh session
			var sessionFormAction string
			listDoc.Find("form#pruefungsterminSearch").Each(func(i int, form *goquery.Selection) {
				action, exists := form.Attr("action")
				if exists {
					sessionFormAction = action
				}
			})

			if sessionFormAction == "" {
				log.Printf("ERROR: No form action in fresh session for exam %d\n", index+1)
				mu.Lock()
				examData[index] = exam // Store basic info only
				mu.Unlock()
				return
			}

			// Find the button for this specific exam in the fresh list page
			buttonName := findButtonForExam(listDoc, exam)
			if buttonName == "" {
				log.Printf("ERROR: Could not find button for exam %d (%s at %s)\n",
					index+1, exam.DateTime, exam.Location)
				mu.Lock()
				examData[index] = exam // Store basic info only
				mu.Unlock()
				return
			}

			// Fetch detail page using the fresh session
			detailDoc, _, err := fetchDetailPage(examClient, listDoc, sessionFormAction, buttonName)
			if err != nil {
				log.Printf("Error fetching detail page for exam %d: %v\n", index+1, err)
				mu.Lock()
				examData[index] = exam // Store basic info only
				mu.Unlock()
				return
			}

			// Parse details into the exam struct
			parseDetailPage(detailDoc, &exam)

			// Store the complete exam data
			mu.Lock()
			examData[index] = exam
			mu.Unlock()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	log.Println("All exam details fetched successfully")

	// Sort the exam data
	sortExamData(examData)

	// Create output
	output := OutputData{
		ExamAppointments: examData,
		TotalCount:       len(examData),
	}

	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal("Error creating JSON:", err)
	}

	fmt.Print(string(jsonOutput))
}
