package pick

import (
	"strings"
	"testing"
)

func TestEscapeAppleScriptPath(t *testing.T) {
	t.Parallel()

	in := `/Users/me/Jobs/Client "A"/Raw\Set`
	got := escapeAppleScriptPath(in)
	want := `/Users/me/Jobs/Client \"A\"/Raw\\Set`
	if got != want {
		t.Fatalf("escapeAppleScriptPath() = %q, want %q", got, want)
	}
}

func TestFolderPickerScript_UsesEscapedPath(t *testing.T) {
	t.Parallel()

	in := `/Users/me/Client "A"`
	got := folderPickerScript(in)

	if want := `POSIX file "/Users/me/Client \"A\""`; !strings.Contains(got, want) {
		t.Fatalf("script missing escaped path; got %q, want fragment %q", got, want)
	}
}
