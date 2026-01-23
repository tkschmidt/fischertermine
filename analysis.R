#!/usr/bin/env Rscript

# Load required libraries
library(data.table)
library(jsonlite)
library(ggplot2)
library(lubridate)

# No longer needed - using scraped_at from JSON

# Find all JSON files in the data directory, excluding latest.json
json_files <- list.files("data", pattern = "exam-data-.*\\.json$", full.names = TRUE)
# Exclude latest.json (symlink)
json_files <- json_files[basename(json_files) != "latest.json"]

# Initialize empty list to store results
all_data <- list()

# Process each JSON file
for (json_file in json_files) {
  cat("Processing:", json_file, "\n")

  # Read JSON file
  data <- tryCatch({
    fromJSON(json_file)
  }, error = function(e) {
    cat("Error reading", json_file, ":", e$message, "\n")
    return(NULL)
  })

  # Skip if data is NULL
  if (is.null(data)) {
    next
  }

  # Check if scraped_at field exists
  if (is.null(data$scraped_at)) {
    cat("Warning: No scraped_at field in", json_file, "\n")
    next
  }

  # Extract scrape date from JSON
  scrape_date <- as.POSIXct(data$scraped_at, format = "%Y-%m-%dT%H:%M:%SZ", tz = "UTC")

  # Skip if no exam appointments
  if (is.null(data$exam_appointments) || length(data$exam_appointments) == 0) {
    cat("Warning: No exam appointments in", json_file, "\n")
    next
  }

  # Convert to data.table
  appointments <- as.data.table(data$exam_appointments)

  # Add scrape date
  appointments[, scrape_date := scrape_date]

  # Calculate free spots
  appointments[, free_spots := as.numeric(max_participants) - as.numeric(current_participants)]

  # Store in list
  all_data[[json_file]] <- appointments
}

# Combine all data
dt <- rbindlist(all_data, fill = TRUE)

# Aggregate by location and scrape date to get total free spots
dt_agg <- dt[, .(
  total_free_spots = sum(free_spots, na.rm = TRUE),
  total_appointments = .N
), by = .(location, scrape_date)]

# Filter to last 3 weeks only
three_weeks_ago <- Sys.time() - weeks(3)
dt_agg <- dt_agg[scrape_date >= three_weeks_ago]

# Sort by scrape date
setorder(dt_agg, scrape_date)

# Print summary
cat("\nData summary:\n")
cat("Total scrapes:", uniqueN(dt_agg$scrape_date), "\n")
cat("Total locations:", uniqueN(dt_agg$location), "\n")
cat("Date range:", format(min(dt_agg$scrape_date), "%Y-%m-%d"), "to", format(max(dt_agg$scrape_date), "%Y-%m-%d"), "\n")

# Create line plot
p <- ggplot(dt_agg, aes(x = scrape_date, y = total_free_spots, color = location, group = location)) +
  geom_line(linewidth = 0.8, alpha = 0.7) +
  geom_point(size = 1.5, alpha = 0.6) +
  labs(
    title = "Free Exam Spots Over Time by Location",
    subtitle = "Bavarian Fishing Examination",
    x = "Scrape Date",
    y = "Number of Free Spots",
    color = "Location"
  ) +
  theme_minimal() +
  theme(
    plot.title = element_text(size = 16, face = "bold"),
    plot.subtitle = element_text(size = 12),
    axis.text.x = element_text(angle = 45, hjust = 1),
    legend.position = "right",
    legend.text = element_text(size = 8),
    panel.grid.minor = element_blank()
  ) +
  scale_x_datetime(date_breaks = "3 days", date_labels = "%b %d")

# Save plot
ggsave("plots/free_spots_by_location.png", p, width = 16, height = 10, dpi = 300)
cat("\nPlot saved to: plots/free_spots_by_location.png\n")

# Create a second plot: top 10 locations by average free spots
top_locations <- dt_agg[, .(avg_free_spots = mean(total_free_spots)), by = location][
  order(-avg_free_spots)
][1:min(10, .N)]

dt_top <- dt_agg[location %in% top_locations$location]

p2 <- ggplot(dt_top, aes(x = scrape_date, y = total_free_spots, color = location, group = location)) +
  geom_line(linewidth = 1, alpha = 0.8) +
  geom_point(size = 2, alpha = 0.7) +
  labs(
    title = "Free Exam Spots Over Time - Top 10 Locations",
    subtitle = "Locations with highest average free spots",
    x = "Scrape Date",
    y = "Number of Free Spots",
    color = "Location"
  ) +
  theme_minimal() +
  theme(
    plot.title = element_text(size = 16, face = "bold"),
    plot.subtitle = element_text(size = 12),
    axis.text.x = element_text(angle = 45, hjust = 1),
    legend.position = "right",
    legend.text = element_text(size = 9),
    panel.grid.minor = element_blank()
  ) +
  scale_x_datetime(date_breaks = "3 days", date_labels = "%b %d")

ggsave("plots/free_spots_top10_locations.png", p2, width = 16, height = 10, dpi = 300)
cat("Plot saved to: plots/free_spots_top10_locations.png\n")

# Create summary table
summary_table <- dt_agg[, .(
  min_free_spots = min(total_free_spots),
  avg_free_spots = round(mean(total_free_spots), 1),
  max_free_spots = max(total_free_spots),
  total_observations = .N
), by = location][order(-avg_free_spots)]

fwrite(summary_table, "plots/location_summary.csv")
cat("Summary table saved to: plots/location_summary.csv\n")

cat("\nTop 10 locations by average free spots:\n")
print(summary_table[1:min(10, .N)])
