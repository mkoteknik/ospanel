package handler

import "testing"

func TestIsValidDomain(t *testing.T) {
  tests := map[string]bool{
    "example.com": true,
    "aura-mock123.com": true,
    "bad..com": false,
    "-bad.com": false,
    "localhost": false,
    "test": false,
    "a.b": false,
  }
  for d, want := range tests {
    if got := isValidDomain(d); got != want {
      t.Errorf("isValidDomain %q = %v want %v", d, got, want)
    }
  }
}
