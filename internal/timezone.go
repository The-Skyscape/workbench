package internal

import (
	"net/http"
	"time"
)

// GetUserTimezone extracts the user's timezone from the request header
func GetUserTimezone(r *http.Request) *time.Location {
	tzHeader := r.Header.Get("X-User-Timezone")
	if tzHeader == "" {
		Log.Debug("No timezone header provided, using UTC")
		return time.UTC // Default to UTC if no timezone provided
	}
	
	Log.Debug("Timezone header received: %s", tzHeader)
	loc, err := time.LoadLocation(tzHeader)
	if err != nil {
		Log.Warn("Invalid timezone %s: %v", tzHeader, err)
		return time.UTC
	}
	
	return loc
}

// FormatTimeInUserTZ formats a time in the user's timezone
func FormatTimeInUserTZ(t time.Time, r *http.Request) string {
	loc := GetUserTimezone(r)
	return t.In(loc).Format("Jan 2, 3:04 PM")
}

// FormatTimeInUserTZLong formats a time with full date in the user's timezone
func FormatTimeInUserTZLong(t time.Time, r *http.Request) string {
	loc := GetUserTimezone(r)
	return t.In(loc).Format("Mon, Jan 2, 2006 at 3:04 PM")
}