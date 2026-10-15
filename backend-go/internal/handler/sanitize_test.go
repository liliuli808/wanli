package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean email unchanged", "a@b.com", "a@b.com"},
		{"trim whitespace", "  a@b.com  ", "a@b.com"},
		{"strip trailing control char", "a@b.com\x01", "a@b.com"},
		{"strip leading control char", "\x00a@b.com", "a@b.com"},
		{"strip mixed controls and whitespace", " \x01a@b.com\x1f ", "a@b.com"},
		{"strip tab", "a@b.com\t", "a@b.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeEmail(tt.input); got != tt.want {
				t.Errorf("sanitizeEmail(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanEmailInJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no email field", `{"password":"x"}`, `{"password":"x"}`},
		{"dirty email cleaned", `{"email":"a@b.com\u0001","password":"x"}`, `{"email":"a@b.com","password":"x"}`},
		{"clean email untouched", `{"email":"a@b.com","password":"x"}`, `{"email":"a@b.com","password":"x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanEmailInJSON([]byte(tt.input))
			if err != nil {
				t.Fatalf("cleanEmailInJSON(%q) error: %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Errorf("cleanEmailInJSON(%q) = %q, want %q", tt.input, string(got), tt.want)
			}
		})
	}
}

func TestPrepareBodyWithCleanEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := bytes.NewBufferString(`{"email":"a@b.com\u0001","password":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	if err := prepareBodyWithCleanEmail(c); err != nil {
		t.Fatalf("prepareBodyWithCleanEmail error: %v", err)
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read body error: %v", err)
	}
	want := `{"email":"a@b.com","password":"x"}`
	if string(raw) != want {
		t.Errorf("cleaned body = %q, want %q", string(raw), want)
	}
}
