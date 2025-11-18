# Bavarian Fishing Exam Data Scraper

Automated scraper for Bavarian fishing examination dates from the official website.

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
- Multiple updates per day get timestamp suffix: `data/exam-data-YYYY-MM-DD-HHMM.json`

## Data Structure

The JSON output contains exam appointments with:
- Date and time
- Exam center location
- City and region
- Availability status (Frei/Belegt)

## Data Analysis

To analyze the historical exam data:

```bash
# Option 1: Use Homebrew R with pre-built packages
brew install r
R -e "install.packages(c('data.table', 'jsonlite', 'rmarkdown'), repos = 'https://cran.rstudio.com')"

# Option 2: Install specific versions that avoid compilation
R -e "install.packages('jsonlite', repos = 'https://cran.rstudio.com'); install.packages('data.table', repos = 'https://cran.rstudio.com'); install.packages('rmarkdown', repos = 'https://cran.rstudio.com')"

# Generate analysis report
Rscript -e "rmarkdown::render('analysis.Rmd')"
```

This creates an HTML report (`analysis.html`) that:
- Combines all JSON files into a single data.table
- Tracks availability changes over time
- Shows exam appointment trends by location and date