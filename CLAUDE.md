# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based web scraper that extracts Bavarian fishing examination data from the official government website (https://fischerpruefung-online.bayern.de/). The scraper runs automated daily updates via GitHub Actions and creates pull requests when data changes.

## Commands

### Development
```bash
# Run the scraper and output JSON to stdout
go run main.go

# Save output to a file
go run main.go > exam-data.json

# Install dependencies
go mod download

# Update dependencies
go mod tidy
```

### Testing the Scraper
```bash
# Test locally and compare with existing data
go run main.go > test-output.json
diff test-output.json data/latest.json
```

## Architecture

### Core Components

**main.go**: Single-file application containing:
- HTTP client with cookie jar for session management
- HTML parsing using goquery library
- Table extraction logic with filtering for exam appointment rows
- JSON output generation

**Data Processing Flow**:
1. Establishes session with base website
2. Fetches exam search page with specific execution parameter
3. Parses HTML tables looking for exam appointment data
4. Filters out header rows and invalid entries
5. Structures data into JSON format with exam appointments

### Data Storage Strategy

- **Versioned Storage**: Files stored as `data/exam-data-YYYY-MM-DD.json`
- **Deduplication**: MD5 hash comparison prevents duplicate data storage
- **Latest Pointer**: `data/latest.json` symlink always points to most recent data
- **Collision Handling**: Same-day updates get timestamp suffix (`data/exam-data-YYYY-MM-DD-HHMM.json`)

### GitHub Actions Automation

**Workflow**: `.github/workflows/update-exams.yml`
- Runs daily at midnight UTC
- Compares new data MD5 with existing latest.json
- Only commits and creates PR if data has changed
- Uses PAT_TOKEN for enhanced permissions if available

## Data Structure

Exam appointments are structured as:
```json
{
  "exam_appointments": [
    {
      "date_time": "Datum, Zeit",
      "location": "Prüfungsort",
      "city": "Stadt",
      "region": "Region",
      "status": "Frei/Belegt"
    }
  ],
  "total_count": number
}
```

## Web Scraping Logic

### Table Detection
- Finds tables with substantial text content (>50 characters)
- Filters out header separator rows containing multiple city/region names
- Validates exam rows by checking for date patterns (DD.MM.YYYY) and time patterns (HH:MM)

### Session Management
- Uses cookie jar for session persistence
- Sets appropriate User-Agent headers
- Handles redirects (up to 20)
- Accesses base page first to establish session

### Content Deduplication
The automation system only commits new JSON files when content actually changes, determined by MD5 hash comparison. This prevents unnecessary commits when the website data hasn't been updated.

## Dependencies

- **github.com/PuerkitoBio/goquery**: HTML parsing and DOM traversal
- **Standard library**: HTTP client, JSON encoding, string manipulation

## Important Notes

- The scraper targets a specific execution URL parameter (`execution=e9s1`)
- Content filtering logic is tuned for Bavarian fishing exam website structure
- The system handles German text encoding and regional formatting
- Error handling focuses on graceful degradation rather than strict validation