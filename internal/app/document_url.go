package app

import (
	"errors"
	"net/url"
	"strings"
)

// validateDocumentURL validates a stored reference without fetching remote content.
// A PDF URL can be signed or extensionless, so its path need not end in .pdf.
func validateDocumentURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed == nil {
		return "", errors.New("file_url must be an absolute HTTP or HTTPS URL")
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
		return "", errors.New("file_url must be an absolute HTTP or HTTPS URL without embedded credentials")
	}
	return raw, nil
}
