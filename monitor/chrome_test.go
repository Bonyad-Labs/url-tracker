package monitor

import (
	"testing"
)

func TestParseActiveTab(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantURL   string
		wantTitle string
		wantErr   bool
	}{
		{
			name:      "Valid output",
			output:    "https://google.com|||Google",
			wantURL:   "https://google.com",
			wantTitle: "Google",
			wantErr:   false,
		},
		{
			name:      "URL with delimiter",
			output:    "https://site.com/search?q=|||Search|||Result",
			wantURL:   "https://site.com/search?q=",
			wantTitle: "Search",
			wantErr:   false,
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
			name:    "Malformed output",
			output:  "no delimiter here",
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
