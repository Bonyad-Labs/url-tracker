package monitor

import (
	"testing"
)

func TestParseActiveTab(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantBrowser string
		wantURL     string
		wantTitle   string
		wantErr     bool
	}{
		{
			name:        "Valid Chrome output",
			output:      "Chrome|||https://google.com|||Google",
			wantBrowser: "Chrome",
			wantURL:     "https://google.com",
			wantTitle:   "Google",
			wantErr:     false,
		},
		{
			name:        "Valid Safari output",
			output:      "Safari|||https://apple.com|||Apple",
			wantBrowser: "Safari",
			wantURL:     "https://apple.com",
			wantTitle:   "Apple",
			wantErr:     false,
		},
		{
			name:        "URL with delimiter",
			output:      "Chrome|||https://site.com/search?q=|||Search Result",
			wantBrowser: "Chrome",
			wantURL:     "https://site.com/search?q=",
			wantTitle:   "Search Result",
			wantErr:     false,
		},
		{
			name:    "Empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "Whitespace output",
			output:  "  \n  ",
			wantErr: true,
		},
		{
			name:    "Malformed output (too few parts)",
			output:  "Chrome|||https://google.com",
			wantErr: true,
		},
		{
			name:    "Missing value output",
			output:  "Safari|||missing value|||Apple",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseActiveTab(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseActiveTab() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Browser != tt.wantBrowser {
					t.Errorf("parseActiveTab() Browser = %v, want %v", got.Browser, tt.wantBrowser)
				}
				if got.URL != tt.wantURL {
					t.Errorf("parseActiveTab() URL = %v, want %v", got.URL, tt.wantURL)
				}
				if got.Title != tt.wantTitle {
					t.Errorf("parseActiveTab() Title = %v, want %v", got.Title, tt.wantTitle)
				}
			}
		})
	}
}
