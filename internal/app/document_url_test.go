package app

import "testing"

func TestDocumentURL(t *testing.T) {
	for _, raw := range []string{"https://example.com/carta.pdf", "https://example.com/download?id=123&signature=abc", " http://localhost:8080/file.pdf "} {
		if _, err := validateDocumentURL(raw); err != nil {
			t.Fatalf("%q: %v", raw, err)
		}
	}
	for _, raw := range []string{"", "/file.pdf", "javascript:alert(1)", "file:///tmp/doc.pdf", "https://user:pass@example.com/doc.pdf", "https://"} {
		if _, err := validateDocumentURL(raw); err == nil {
			t.Fatalf("invalid URL accepted: %q", raw)
		}
	}
}
