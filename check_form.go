package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	baseURL := "https://fischerpruefung-online.bayern.de/fprApp/"
	listURL := "https://fischerpruefung-online.bayern.de/fprApp/verwaltung/Pruefungssuche?execution=e9s1"

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Visit base page
	req, _ := http.NewRequest("GET", baseURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client.Do(req)

	// Get list page
	req, _ = http.NewRequest("GET", listURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(resp.Body)

	// Find all forms
	doc.Find("form").Each(func(i int, form *goquery.Selection) {
		action, _ := form.Attr("action")
		method, _ := form.Attr("method")
		id, _ := form.Attr("id")
		enctype, _ := form.Attr("enctype")

		fmt.Printf("\n=== Form %d ===\n", i)
		fmt.Printf("ID: %s\n", id)
		fmt.Printf("Action: %s\n", action)
		fmt.Printf("Method: %s\n", method)
		fmt.Printf("Enctype: %s\n", enctype)

		// Count submit buttons in this form
		submitCount := 0
		form.Find("input[type=submit]").Each(func(j int, s *goquery.Selection) {
			submitCount++
		})
		fmt.Printf("Submit buttons: %d\n", submitCount)
	})
}
