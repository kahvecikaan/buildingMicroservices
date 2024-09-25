package data

import "testing"

func TestChecksValidation(t *testing.T) {
	testCases := []struct {
		name  string
		sku   string
		valid bool
	}{
		{"Valid SKU", "abc-abc-abc", true},
		{"Invalid SKU - Spaces", "abc-abc-abc xts-adc", false},
		{"Valid SKU", "abc-abc-abcd", true},
		{"Invalid SKU - Uppercase", "ABC-abc-abc", false},
		{"Invalid SKU - Missing Hyphen", "abcabc-abc", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &Product{
				Name:  "test",
				Price: 1,
				SKU:   tc.sku,
			}

			err := p.Validate()

			if tc.valid && err != nil {
				t.Fatalf("Expected valid SKU, got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Fatalf("Expected invalid SKU, got no error")
			}
		})
	}
}
