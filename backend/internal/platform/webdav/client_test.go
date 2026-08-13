package webdav

import "testing"

func TestSafePathRejectsTraversal(t *testing.T) {
	for _, value := range []string{"../secret", "employees/../../secret", `employees\..\secret`, "/absolute"} {
		if _, err := safePath(value); err == nil {
			t.Fatalf("safePath(%q) accepted traversal", value)
		}
	}
}

func TestSafePathAcceptsNestedRelativePath(t *testing.T) {
	got, err := safePath("employees/123/contract.pdf")
	if err != nil {
		t.Fatalf("safePath() error = %v", err)
	}
	if got != "employees/123/contract.pdf" {
		t.Fatalf("safePath() = %q", got)
	}
}
