package logger

import (
	"strings"
	"unicode"
)

// SanitizeForLog removes control characters and newlines from log input to prevent injection
func SanitizeForLog(input string) string {
	if input == "" {
		return ""
	}

	// Remove all control characters including newlines, tabs, etc.
	sanitized := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, input)

	// Limit length to prevent excessive log sizes
	const maxLogLength = 1000
	if len(sanitized) > maxLogLength {
		sanitized = sanitized[:maxLogLength] + "..."
	}

	return sanitized
}

// SanitizeIP sanitizes IP addresses for logging, preserving the format
func SanitizeIP(ip string) string {
	// Remove any control characters but preserve the IP format
	parts := strings.Split(ip, ":")
	if len(parts) > 0 {
		// Sanitize the IP part
		parts[0] = SanitizeForLog(parts[0])
	}
	if len(parts) > 1 {
		// Sanitize the port part
		parts[1] = SanitizeForLog(parts[1])
	}
	return strings.Join(parts, ":")
}

// MaskSensitive masks sensitive data like client IDs or solutions
func MaskSensitive(input string) string {
	if len(input) <= 8 {
		return "***"
	}
	// Show first 4 and last 4 characters
	return input[:4] + "..." + input[len(input)-4:]
}