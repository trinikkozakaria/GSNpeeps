package objectpath

import "testing"

func TestSafePathRejectsTraversal(t *testing.T) {
	for _, value := range []string{"../secret", "employees/../../secret", `employees\..\secret`, "/absolute"} {
		if _, err := SafePath(value); err == nil {
			t.Fatalf("SafePath(%q) accepted traversal", value)
		}
	}
}

func TestSafePathRejectsEmpty(t *testing.T) {
	if _, err := SafePath("   "); err == nil {
		t.Fatal("SafePath(whitespace) accepted an empty path")
	}
}

func TestSafePathAcceptsNestedRelativePath(t *testing.T) {
	got, err := SafePath("employees/123/contract.pdf")
	if err != nil {
		t.Fatalf("SafePath() error = %v", err)
	}
	if got != "employees/123/contract.pdf" {
		t.Fatalf("SafePath() = %q", got)
	}
}
