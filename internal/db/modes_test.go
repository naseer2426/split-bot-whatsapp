package db

import "testing"

func TestValidateChatMetaMode(t *testing.T) {
	t.Parallel()

	valid := []string{"silent", "splitbot", "nanobot", "hermes", "playground"}
	for _, mode := range valid {
		if err := ValidateChatMetaMode(mode); err != nil {
			t.Fatalf("ValidateChatMetaMode(%q) = %v, want nil", mode, err)
		}
	}

	if err := ValidateChatMetaMode("unknown"); err == nil {
		t.Fatal(`ValidateChatMetaMode("unknown") = nil, want error`)
	}
}
