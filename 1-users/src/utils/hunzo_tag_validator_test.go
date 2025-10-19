package utils

import (
	"testing"
)

func TestValidateOmniTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantErr bool
	}{
		{
			name:    "Valid tag - alphanumeric 5 chars",
			tag:     "Jr234",
			wantErr: false,
		},
		{
			name:    "Valid tag - all letters",
			tag:     "ABCDE",
			wantErr: false,
		},
		{
			name:    "Valid tag - all numbers",
			tag:     "12345",
			wantErr: false,
		},
		{
			name:    "Valid tag - mixed case",
			tag:     "Abc12",
			wantErr: false,
		},
		{
			name:    "Valid tag - 1 character",
			tag:     "A",
			wantErr: false,
		},
		{
			name:    "Invalid tag - too long (6 chars)",
			tag:     "Jr2345",
			wantErr: true,
		},
		{
			name:    "Invalid tag - empty",
			tag:     "",
			wantErr: true,
		},
		{
			name:    "Invalid tag - special characters",
			tag:     "Jr@23",
			wantErr: true,
		},
		{
			name:    "Invalid tag - space",
			tag:     "Jr 23",
			wantErr: true,
		},
		{
			name:    "Invalid tag - underscore",
			tag:     "Jr_23",
			wantErr: true,
		},
		{
			name:    "Invalid tag - dash",
			tag:     "Jr-23",
			wantErr: true,
		},
		{
			name:    "Valid tag - case sensitive (lowercase)",
			tag:     "jr234",
			wantErr: false,
		},
		{
			name:    "Valid tag - case sensitive (uppercase)",
			tag:     "JR234",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOmniTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOmniTag() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
