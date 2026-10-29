package app

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestSwagger(t *testing.T) {
	r := (&App{}).Router()
	for _, path := range []string{"/swagger/", "/swagger/index.html", "/swagger/openapi.json", "/swagger/swagger-ui.css", "/swagger/swagger-ui-bundle.js"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	var spec struct {
		OpenAPI string
		Paths   map[string]json.RawMessage
	}
	if err := json.Unmarshal(openAPISpec, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.OpenAPI != "3.0.3" {
		t.Fatal("invalid OpenAPI version")
	}
	for _, path := range []string{"/api/v1/departments", "/api/v1/users", "/api/v1/login"} {
		if _, ok := spec.Paths[path]; !ok {
			t.Fatalf("missing endpoint %s", path)
		}
	}
}
