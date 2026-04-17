package i18n

import "testing"

func TestT_InternalFormattingAndFallback(t *testing.T) {
	original := generalMessages[En]["welcome"]
	generalMessages[En]["welcome"] = "hello %s"
	t.Cleanup(func() {
		if original == "" {
			delete(generalMessages[En], "welcome")
			return
		}
		generalMessages[En]["welcome"] = original
	})

	if got := T(En, "welcome", "alice"); got != "hello alice" {
		t.Fatalf("T formatting = %q, want %q", got, "hello alice")
	}
	if got := T(Zh, "welcome", "alice"); got != "hello alice" {
		t.Fatalf("expected zh fallback to english formatted message, got %q", got)
	}
}
