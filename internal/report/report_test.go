package report

import "testing"

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{25623142, "24.44 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		result := humanBytes(tt.input)
		if result != tt.expected {
			t.Errorf("humanBytes(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}