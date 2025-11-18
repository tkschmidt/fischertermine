# Bavarian Fishing Exam Data Scraper

Automated scraper for Bavarian fishing examination dates from the official website. A report can be seen at on [https://tkschmidt.github.io/fischertermine](https://tkschmidt.github.io/fischertermine/).

## Features

- Scrapes exam data from https://fischerpruefung-online.bayern.de/
- Outputs structured JSON with dates, locations, and availability
- Automated daily updates via GitHub Actions
- Creates pull requests when data changes

## Usage

```bash
go run main.go > exam-data.json
```

## Data Storage

- Exam data is stored in the `data/` folder with date-based filenames: `data/exam-data-YYYY-MM-DD.json`
- `data/latest.json` symlink always points to the most recent data
- Only stores new files when MD5 hash changes (deduplication)

## Data Structure

The JSON output contains exam appointments with:
- Date and time
- Exam center location
- City and region
- Availability status (Frei/Belegt)

