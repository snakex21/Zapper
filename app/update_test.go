package main

import "testing"

func TestVersionNewer(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"10.3.1", "10.3.0", true},
		{"10.4.0", "10.3.9", true},
		{"11.0.0", "10.99.99", true},
		{"10.3.0", "10.3.0", false},
		{"10.2.9", "10.3.0", false},
		{"v10.3.1", "10.3.0", true},
	}
	for _, test := range tests {
		got, err := versionNewer(test.candidate, test.current)
		if err != nil {
			t.Fatalf("versionNewer(%q, %q): %v", test.candidate, test.current, err)
		}
		if got != test.want {
			t.Fatalf("versionNewer(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}

func TestParseSHA256File(t *testing.T) {
	name := "Zapper-v10.4.0-Windows-x64.zip"
	hash := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, err := parseSHA256File(hash+"  "+name+"\n", name)
	if err != nil {
		t.Fatal(err)
	}
	if got != hash {
		t.Fatalf("got %q, want %q", got, hash)
	}
}
