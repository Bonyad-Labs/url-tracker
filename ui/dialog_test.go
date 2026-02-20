package ui

import (
	"testing"
)

func TestQuoteForAppleScript(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", `"Hello"`},
		{`Hello "World"`, `"Hello \"World\""`},
		{"Line 1\nLine 2", `"Line 1\nLine 2"`},
		{`Slash \ Test`, `"Slash \\ Test"`},
	}

	for _, tt := range tests {
		got := quoteForAppleScript(tt.input)
		if got != tt.expected {
			t.Errorf("quoteForAppleScript(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractTextAndButtonFromDialog(t *testing.T) {
	tests := []struct {
		input      string
		wantText   string
		wantButton string
	}{
		{
			input:      "button returned:OK, text returned:user input",
			wantText:   "user input",
			wantButton: "OK",
		},
		{
			input:      "button returned:Cancel, text returned:",
			wantButton: "Cancel",
			wantText:   "",
		},
		{
			input:      "button returned:Save URL, text returned:my site",
			wantButton: "Save URL",
			wantText:   "my site",
		},
	}

	for _, tt := range tests {
		gotText, gotButton := extractTextAndButtonFromDialog(tt.input)
		if gotText != tt.wantText {
			t.Errorf("extractTextAndButtonFromDialog(%q) gotText = %q, want %q", tt.input, gotText, tt.wantText)
		}
		if gotButton != tt.wantButton {
			t.Errorf("extractTextAndButtonFromDialog(%q) gotButton = %q, want %q", tt.input, gotButton, tt.wantButton)
		}
	}
}
