package utils

import (
	"math/big"
	"testing"
)

func TestFormatHostCount(t *testing.T) {
	tests := []struct {
		input    *big.Int
		expected string
	}{
		{nil, "0"},
		{big.NewInt(0), "0"},
		{big.NewInt(-5), "0"},
		{big.NewInt(1), "1"},
		{big.NewInt(25), "25"},
		{big.NewInt(500), "500"},
		{big.NewInt(999999), "999999"},
		{big.NewInt(1000000), "1M+"},
		{big.NewInt(1544192), "1M+"},
		{big.NewInt(9999999), "9M+"},
		{big.NewInt(10000000), "10M+"},
		{big.NewInt(18500000), "10M+"},
		{big.NewInt(25000000), "20M+"},
		{big.NewInt(99000000), "90M+"},
		{big.NewInt(100000000), "100M+"},
		{big.NewInt(1200000000), "1B+"},
		{big.NewInt(25000000000), "20B+"},
	}

	for _, tt := range tests {
		got := FormatHostCount(tt.input)
		if got != tt.expected {
			t.Errorf("FormatHostCount(%v) = %q, expected %q", tt.input, got, tt.expected)
		}
	}

	// Test huge IPv6 range
	huge := new(big.Int).Exp(big.NewInt(2), big.NewInt(64), nil)
	gotHuge := FormatHostCount(huge)
	if gotHuge != "10M+" {
		t.Errorf("FormatHostCount(2^64) = %q, expected %q", gotHuge, "10M+")
	}
}
