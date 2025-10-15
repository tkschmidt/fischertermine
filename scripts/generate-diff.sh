#!/bin/bash

# Script to generate semantic diff between exam data files
# Usage: ./generate-diff.sh <old_file> <new_file>

set -e

if [ $# -ne 2 ]; then
    echo "Usage: $0 <old_file> <new_file>"
    exit 1
fi

OLD_FILE="$1"
NEW_FILE="$2"

if [ ! -f "$OLD_FILE" ]; then
    echo "📊 Initial data load - no previous version to compare"
    echo ""
    echo "This is the first time exam data is being added to the repository."
    exit 0
fi

if [ ! -f "$NEW_FILE" ]; then
    echo "Error: New file $NEW_FILE does not exist"
    exit 1
fi

# Check if files are actually different
OLD_MD5=$(md5sum "$OLD_FILE" | cut -d' ' -f1)
NEW_MD5=$(md5sum "$NEW_FILE" | cut -d' ' -f1)

echo "Debug: Old MD5: $OLD_MD5" >&2
echo "Debug: New MD5: $NEW_MD5" >&2

if [ "$OLD_MD5" = "$NEW_MD5" ]; then
    echo "No changes detected between files"
    exit 0
fi

echo "📊 Changes detected between versions:"
echo ""

# Create temp files
OLD_TEMP=$(mktemp)
NEW_TEMP=$(mktemp)

# Extract exam appointments and create comparable format
jq -r '.[0].exam_appointments[] | "\(.date_time) | \(.location) | \(.city) | \(.status)"' "$OLD_FILE" | sort > "$OLD_TEMP"
jq -r '.[0].exam_appointments[] | "\(.date_time) | \(.location) | \(.city) | \(.status)"' "$NEW_FILE" | sort > "$NEW_TEMP"

echo "Debug: Old file has $(wc -l < "$OLD_TEMP") appointments" >&2
echo "Debug: New file has $(wc -l < "$NEW_TEMP") appointments" >&2

# Find new appointments
NEW_COUNT=$(comm -13 "$OLD_TEMP" "$NEW_TEMP" | wc -l)
if [ "$NEW_COUNT" -gt 0 ]; then
    echo "🆕 **New appointments ($NEW_COUNT):**"
    comm -13 "$OLD_TEMP" "$NEW_TEMP" | head -10
    if [ "$NEW_COUNT" -gt 10 ]; then
        echo "... and $((NEW_COUNT - 10)) more"
    fi
    echo ""
fi

# Find removed appointments
REMOVED_COUNT=$(comm -23 "$OLD_TEMP" "$NEW_TEMP" | wc -l)
if [ "$REMOVED_COUNT" -gt 0 ]; then
    echo "❌ **Removed appointments ($REMOVED_COUNT):**"
    comm -23 "$OLD_TEMP" "$NEW_TEMP" | head -10
    if [ "$REMOVED_COUNT" -gt 10 ]; then
        echo "... and $((REMOVED_COUNT - 10)) more"
    fi
    echo ""
fi

# Find status changes (same appointment, different status)
echo "🔄 **Status changes:**"

# Simple approach: find appointments that exist in both but with different status
while IFS='|' read -r datetime location city old_status; do
    # Look for same appointment in new file
    new_status=$(grep -F "$datetime | $location | $city |" "$NEW_TEMP" | cut -d'|' -f4 | sed 's/^ *//; s/ *$//')
    if [ -n "$new_status" ] && [ "$old_status" != "$new_status" ]; then
        echo "$datetime | $location | $city: $old_status → $new_status"
    fi
done < "$OLD_TEMP" | head -10

# Clean up temp files
rm -f "$OLD_TEMP" "$NEW_TEMP" "${OLD_TEMP}".* "${NEW_TEMP}".*

echo ""
echo "Summary: $NEW_COUNT new, $REMOVED_COUNT removed appointments"