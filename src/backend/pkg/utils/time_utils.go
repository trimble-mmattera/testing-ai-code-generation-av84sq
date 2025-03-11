// Package utils provides time utility functions for the Document Management Platform.
// This file contains utilities for time formatting, parsing, comparison, and other
// time-related helper functions used across the system.
package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"../errors"
)

// Time format constants
const (
	DefaultTimeFormat     = time.RFC3339
	DefaultDateFormat     = "2006-01-02"
	DefaultDateTimeFormat = "2006-01-02 15:04:05"
	DefaultDurationFormat = "24h"
)

// FormatTime formats a time.Time value using the specified format
func FormatTime(t time.Time, format string) string {
	if format == "" {
		format = DefaultTimeFormat
	}
	return t.Format(format)
}

// FormatTimeDefault formats a time.Time value using the default time format (RFC3339)
func FormatTimeDefault(t time.Time) string {
	return FormatTime(t, DefaultTimeFormat)
}

// FormatDate formats a time.Time value as a date string (YYYY-MM-DD)
func FormatDate(t time.Time) string {
	return FormatTime(t, DefaultDateFormat)
}

// FormatDateTime formats a time.Time value as a date and time string (YYYY-MM-DD HH:MM:SS)
func FormatDateTime(t time.Time) string {
	return FormatTime(t, DefaultDateTimeFormat)
}

// ParseTime parses a time string using the specified format
func ParseTime(timeStr string, format string) (time.Time, error) {
	if format == "" {
		format = DefaultTimeFormat
	}
	
	t, err := time.Parse(format, timeStr)
	if err != nil {
		return time.Time{}, errors.NewValidationError("invalid time format: " + err.Error())
	}
	
	return t, nil
}

// ParseTimeDefault parses a time string using the default time format (RFC3339)
func ParseTimeDefault(timeStr string) (time.Time, error) {
	return ParseTime(timeStr, DefaultTimeFormat)
}

// ParseDate parses a date string (YYYY-MM-DD) into a time.Time value
func ParseDate(dateStr string) (time.Time, error) {
	return ParseTime(dateStr, DefaultDateFormat)
}

// ParseDateTime parses a date and time string (YYYY-MM-DD HH:MM:SS) into a time.Time value
func ParseDateTime(dateTimeStr string) (time.Time, error) {
	return ParseTime(dateTimeStr, DefaultDateTimeFormat)
}

// Now returns the current time in UTC
func Now() time.Time {
	return time.Now().UTC()
}

// Today returns the current date at midnight UTC
func Today() time.Time {
	now := Now()
	return now.Truncate(24 * time.Hour)
}

// IsToday checks if a time.Time value is today
func IsToday(t time.Time) bool {
	today := Today()
	tDay := t.Truncate(24 * time.Hour)
	return today.Equal(tDay)
}

// IsFuture checks if a time.Time value is in the future
func IsFuture(t time.Time) bool {
	now := Now()
	return t.After(now)
}

// IsPast checks if a time.Time value is in the past
func IsPast(t time.Time) bool {
	now := Now()
	return t.Before(now)
}

// AddDays adds a specified number of days to a time.Time value
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddMonths adds a specified number of months to a time.Time value
func AddMonths(t time.Time, months int) time.Time {
	return t.AddDate(0, months, 0)
}

// AddYears adds a specified number of years to a time.Time value
func AddYears(t time.Time, years int) time.Time {
	return t.AddDate(years, 0, 0)
}

// DaysBetween calculates the number of days between two time.Time values
func DaysBetween(t1, t2 time.Time) int {
	// Truncate to days to ignore partial days
	t1Day := t1.Truncate(24 * time.Hour)
	t2Day := t2.Truncate(24 * time.Hour)
	
	// Calculate the absolute days between
	days := int(math.Abs(t2Day.Sub(t1Day).Hours() / 24))
	return days
}

// MonthsBetween calculates the approximate number of months between two time.Time values
func MonthsBetween(t1, t2 time.Time) int {
	// Calculate years and months difference
	yearDiff := t2.Year() - t1.Year()
	monthDiff := int(t2.Month()) - int(t1.Month())
	
	// Total months
	months := yearDiff*12 + monthDiff
	
	// Adjust for day of month if needed
	if t2.Day() < t1.Day() && months > 0 {
		months--
	} else if t2.Day() > t1.Day() && months < 0 {
		months++
	}
	
	return int(math.Abs(float64(months)))
}

// YearsBetween calculates the approximate number of years between two time.Time values
func YearsBetween(t1, t2 time.Time) int {
	months := MonthsBetween(t1, t2)
	return months / 12
}

// ParseDuration parses a duration string into a time.Duration value
func ParseDuration(durationStr string) (time.Duration, error) {
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0, errors.NewValidationError("invalid duration format: " + err.Error())
	}
	return d, nil
}

// FormatDuration formats a time.Duration value as a string
func FormatDuration(d time.Duration) string {
	return d.String()
}

// TimeAgo returns a human-readable string representing time elapsed since the given time
func TimeAgo(t time.Time) string {
	now := Now()
	duration := now.Sub(t)
	
	// Convert to appropriate unit
	if duration < time.Minute {
		seconds := int(duration.Seconds())
		if seconds < 5 {
			return "just now"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		return fmt.Sprintf("%d months ago", months)
	}
	
	years := int(duration.Hours() / 24 / 365)
	return fmt.Sprintf("%d years ago", years)
}

// StartOfDay returns the start of the day (midnight) for a given time.Time value
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59.999999999) for a given time.Time value
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfMonth returns the start of the month for a given time.Time value
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the end of the month for a given time.Time value
func EndOfMonth(t time.Time) time.Time {
	// Get the first day of next month
	firstOfNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
	
	// Subtract one nanosecond to get the end of current month
	return firstOfNextMonth.Add(-time.Nanosecond)
}

// StartOfYear returns the start of the year for a given time.Time value
func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// EndOfYear returns the end of the year for a given time.Time value
func EndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
}

// IsExpired checks if a time has expired based on a specified duration
func IsExpired(t time.Time, duration time.Duration) bool {
	now := Now()
	expirationTime := t.Add(duration)
	return now.After(expirationTime)
}

// TimeUntil calculates the duration until a future time
func TimeUntil(t time.Time) (time.Duration, error) {
	now := Now()
	if t.Before(now) {
		return 0, errors.NewValidationError("time is in the past")
	}
	
	duration := t.Sub(now)
	return duration, nil
}

// TimeSince calculates the duration since a past time
func TimeSince(t time.Time) (time.Duration, error) {
	now := Now()
	if t.After(now) {
		return 0, errors.NewValidationError("time is in the future")
	}
	
	duration := now.Sub(t)
	return duration, nil
}