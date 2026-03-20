package launcher

import (
	"os"
	"strings"
	"testing"
)

type recordedCommand struct {
	name string
	args []string
}

func TestLaunchWith_SystemDefault_UsesOpenWithCommandScript(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Default",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     "/Volumes/NIKON Z 9",
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "open" {
		t.Fatalf("command name = %q, want %q", got.name, "open")
	}
	if len(got.args) != 1 {
		t.Fatalf("args = %v, want exactly one script path", got.args)
	}
	scriptPath := got.args[0]
	t.Cleanup(func() { _ = os.Remove(scriptPath) })
	if !strings.HasSuffix(scriptPath, ".command") {
		t.Fatalf("script path = %q, want .command suffix", scriptPath)
	}
	scriptBytes, readErr := os.ReadFile(scriptPath)
	if readErr != nil {
		t.Fatalf("read script %q: %v", scriptPath, readErr)
	}
	script := string(scriptBytes)
	if !strings.Contains(script, "/usr/local/bin/cardbot") {
		t.Fatalf("script missing cardbot path: %s", script)
	}
	if !strings.Contains(script, "/Volumes/NIKON Z 9") {
		t.Fatalf("script missing mount path: %s", script)
	}
}

func TestLaunchWith_TerminalApp_UsesAppleScript(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Terminal",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     "/Volumes/NIKON Z 9",
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "osascript" {
		t.Fatalf("command name = %q, want %q", got.name, "osascript")
	}
	joined := strings.Join(got.args, " ")
	if !strings.Contains(joined, "tell application \"Terminal\" to do script") {
		t.Fatalf("args missing Terminal script command: %v", got.args)
	}
}

func TestLaunchWith_GhosttyDefault_UsesOpenWithE(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Ghostty",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     "/Volumes/CARD",
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "open" {
		t.Fatalf("command name = %q, want %q", got.name, "open")
	}
	if len(got.args) != 7 {
		t.Fatalf("args = %v, want 7 args", got.args)
	}
	if got.args[0] != "-na" || got.args[1] != "Ghostty" {
		t.Fatalf("args = %v, want '-na Ghostty ...'", got.args)
	}
	if got.args[2] != "--args" || !strings.HasPrefix(got.args[3], "--working-directory=") || got.args[4] != "-e" {
		t.Fatalf("args = %v, want '--args --working-directory=<home> -e ...'", got.args)
	}
	if got.args[5] != "/usr/local/bin/cardbot" || got.args[6] != "/Volumes/CARD" {
		t.Fatalf("args = %v, want binary + mount path passed separately", got.args)
	}
}

func TestLaunchWith_GhosttyDefault_PreservesTrailingSpacesInMountPath(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	mount := "/Volumes/NIKON Z 9  "
	err := launchWith(Options{
		TerminalApp:   "Ghostty",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     mount,
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.args[6] != mount {
		t.Fatalf("mount arg = %q, want %q", got.args[6], mount)
	}
}

func TestLaunchWith_CustomLaunchArgs_TemplatesResolved(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Ghostty",
		CardBotBinary: "/opt/cardbot",
		MountPath:     "/Volumes/NIKON Z 9",
		LaunchArgs: []string{
			"-e",
			"{{cardbot_binary}} {{mount_path}}",
		},
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "open" {
		t.Fatalf("command name = %q, want %q", got.name, "open")
	}
	if len(got.args) != 6 {
		t.Fatalf("args = %v, want 6 args", got.args)
	}
	if got.args[0] != "-na" || got.args[1] != "Ghostty" {
		t.Fatalf("args = %v, want '-na Ghostty ...'", got.args)
	}
	if got.args[3] != "-e" || got.args[4] != "/opt/cardbot" || got.args[5] != "/Volumes/NIKON Z 9" {
		t.Fatalf("args = %v, want '--args -e /opt/cardbot /Volumes/NIKON Z 9'", got.args)
	}
}

func TestLaunchWith_GhosttyCustomLaunchArgs_StripsQuotedPlaceholders(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Ghostty",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     "/Volumes/NIKON Z 9",
		LaunchArgs: []string{
			"-e",
			"'{{cardbot_binary}}'",
			"'{{mount_path}}'",
		},
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "open" {
		t.Fatalf("command name = %q, want %q", got.name, "open")
	}
	if got.args[4] != "/usr/local/bin/cardbot" || got.args[5] != "/Volumes/NIKON Z 9" {
		t.Fatalf("args = %v, expected unquoted binary and mount path", got.args)
	}
}

func TestLaunchWith_GhosttyCustomLaunchArgs_LegacyCombinedCommandIsSplit(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Ghostty",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     "/Volumes/NIKON Z 9",
		LaunchArgs: []string{
			"-e",
			"'{{cardbot_binary}}' '{{mount_path}}'",
		},
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "open" {
		t.Fatalf("command name = %q, want %q", got.name, "open")
	}
	if len(got.args) != 6 {
		t.Fatalf("args = %v, want 6 args", got.args)
	}
	if got.args[3] != "-e" || got.args[4] != "/usr/local/bin/cardbot" || got.args[5] != "/Volumes/NIKON Z 9" {
		t.Fatalf("args = %v, expected legacy combined command to split into binary + mount", got.args)
	}
}

func TestLaunchWith_StripsMatchingQuotesFromBinaryAndMount(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "Ghostty",
		CardBotBinary: "'/usr/local/bin/cardbot'",
		MountPath:     "'/Volumes/NIKON Z 9'",
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.args[5] != "/usr/local/bin/cardbot" || got.args[6] != "/Volumes/NIKON Z 9" {
		t.Fatalf("args = %v, expected normalized binary + mount path", got.args)
	}
}

func TestLaunchWith_EmptyTerminalApp_DefaultsToSystemDefault(t *testing.T) {
	var got recordedCommand
	run := func(name string, args ...string) error {
		got = recordedCommand{name: name, args: append([]string{}, args...)}
		return nil
	}

	err := launchWith(Options{
		TerminalApp:   "   ",
		CardBotBinary: "/usr/local/bin/cardbot",
		MountPath:     "/Volumes/CARD",
	}, run)
	if err != nil {
		t.Fatalf("launchWith error: %v", err)
	}
	if got.name != "open" {
		t.Fatalf("command name = %q, want %q", got.name, "open")
	}
	if len(got.args) != 1 || !strings.HasSuffix(got.args[0], ".command") {
		t.Fatalf("args = %v, want one .command path", got.args)
	}
	t.Cleanup(func() { _ = os.Remove(got.args[0]) })
}

func TestLaunchWith_RequiresBinaryAndMountPath(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{name: "missing binary", opts: Options{MountPath: "/Volumes/CARD"}},
		{name: "missing mount path", opts: Options{CardBotBinary: "/usr/local/bin/cardbot"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := launchWith(tt.opts, func(name string, args ...string) error { return nil })
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
