package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// Options controls how a terminal is launched for a detected card.
type Options struct {
	TerminalApp   string
	LaunchArgs    []string
	CardBotBinary string
	MountPath     string
	Debugf        func(format string, args ...any)
	Logf          func(format string, args ...any)
}

type commandRunner func(name string, args ...string) error

// Launch opens the configured terminal and runs cardbot for the given mount path.
func Launch(opts Options) error {
	return launchWith(opts, runCommand)
}

func launchWith(opts Options, run commandRunner) error {
	binary := stripMatchingQuotes(strings.TrimSpace(opts.CardBotBinary))
	mountPath := stripMatchingQuotes(opts.MountPath)
	if binary == "" {
		return fmt.Errorf("cardbot binary path is required")
	}
	if strings.TrimSpace(mountPath) == "" {
		return fmt.Errorf("mount path is required")
	}

	debugf := opts.Debugf
	if debugf == nil {
		debugf = func(string, ...any) {}
	}
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	runLogged := func(name string, args ...string) error {
		formatted := formatCommandArgs(args)
		debugf("exec: %s %s", name, formatted)
		logf("Launcher exec: %s %s", name, formatted)
		return run(name, args...)
	}

	app := normalizeTerminalApp(opts.TerminalApp)
	debugf("launcher config: app=%q binary=%q mount=%q custom_args=%d", app, binary, mountPath, len(opts.LaunchArgs))

	if len(opts.LaunchArgs) > 0 {
		resolved := resolveLaunchArgs(opts.LaunchArgs, binary, mountPath)
		if isGhosttyApp(app) {
			resolved = normalizeGhosttyLaunchArgs(resolved, binary, mountPath)
		}
		debugf("launcher branch: custom launch args")
		if isSystemDefaultTerminal(app) {
			return runLogged("open", resolved...)
		}
		openAppFlag := "-a"
		if isGhosttyApp(app) {
			// Ghostty on macOS requires -n to reliably open a fresh terminal window
			// when Ghostty is already running.
			openAppFlag = "-na"
		}
		openArgs := append([]string{openAppFlag, app, "--args"}, resolved...)
		return runLogged("open", openArgs...)
	}

	if isSystemDefaultTerminal(app) {
		debugf("launcher branch: system default terminal")
		scriptPath, err := writeDefaultTerminalCommandScript(binary, mountPath)
		if err != nil {
			return err
		}
		debugf("generated command script: %s", scriptPath)
		return runLogged("open", scriptPath)
	}

	if isTerminalApp(app) {
		debugf("launcher branch: Terminal AppleScript")
		cmd := fmt.Sprintf("%s %s", shQuote(binary), shQuote(mountPath))
		return runLogged("osascript",
			"-e", fmt.Sprintf(`tell application "Terminal" to do script %q`, cmd),
			"-e", `activate application "Terminal"`,
		)
	}

	if isGhosttyApp(app) {
		debugf("launcher branch: Ghostty")
		// Ghostty expects command and argv as separate arguments after -e.
		// Passing a single shell-quoted string causes it to look for a binary
		// whose name includes spaces (e.g. "/usr/local/bin/cardbot /Volumes/...").
		// Use -na to force a fresh window when Ghostty is already running.
		args := []string{"-na", app, "--args"}
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			// Daemon processes launched by launchd often inherit cwd="/".
			// Set Ghostty's working directory explicitly so new windows don't land at root.
			args = append(args, "--working-directory="+home)
		}
		args = append(args, "-e", binary, mountPath)
		return runLogged("open", args...)
	}

	debugf("launcher branch: generic app")
	return runLogged("open", "-a", app, "--args", binary, mountPath)
}

func normalizeTerminalApp(app string) string {
	app = strings.TrimSpace(app)
	if app == "" {
		return "Default"
	}
	if strings.EqualFold(app, "terminal.app") {
		return "Terminal"
	}
	if strings.EqualFold(app, "default") || strings.EqualFold(app, "system default") || strings.EqualFold(app, "macos default") {
		return "Default"
	}
	if strings.EqualFold(app, "ghostty") {
		return "Ghostty"
	}
	return app
}

func resolveLaunchArgs(args []string, binary, mountPath string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		replaced := strings.ReplaceAll(arg, "{{mount_path}}", mountPath)
		replaced = strings.ReplaceAll(replaced, "{{cardbot_binary}}", binary)
		out = append(out, replaced)
	}
	return out
}

func normalizeGhosttyLaunchArgs(args []string, binary, mountPath string) []string {
	if len(args) == 0 {
		return args
	}

	out := make([]string, 0, len(args)+1)
	for _, arg := range args {
		out = append(out, stripMatchingQuotes(arg))
	}

	for i := 0; i < len(out)-1; i++ {
		if out[i] != "-e" {
			continue
		}
		cmd := strings.TrimSpace(out[i+1])
		if cmd == "" {
			break
		}

		if strings.HasPrefix(cmd, binary+" ") && strings.TrimSpace(strings.TrimPrefix(cmd, binary)) == mountPath {
			replacement := []string{"-e", binary, mountPath}
			prefix := append([]string{}, out[:i]...)
			suffix := append([]string{}, out[i+2:]...)
			return append(append(prefix, replacement...), suffix...)
		}

		if words, ok := parseSimpleShellWords(cmd); ok && len(words) == 2 && words[0] == binary && words[1] == mountPath {
			replacement := []string{"-e", binary, mountPath}
			prefix := append([]string{}, out[:i]...)
			suffix := append([]string{}, out[i+2:]...)
			return append(append(prefix, replacement...), suffix...)
		}

		break
	}

	return out
}

func stripMatchingQuotes(s string) string {
	if len(s) < 2 {
		return s
	}

	if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		inner := s[1 : len(s)-1]
		if !strings.Contains(inner, "'") {
			return inner
		}
		return s
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		inner := s[1 : len(s)-1]
		if !strings.Contains(inner, "\"") {
			return inner
		}
		return s
	}
	return s
}

func parseSimpleShellWords(s string) ([]string, bool) {
	var words []string
	var token strings.Builder
	inSingle := false
	inDouble := false
	escape := false
	hasToken := false

	flush := func() {
		if !hasToken {
			return
		}
		words = append(words, token.String())
		token.Reset()
		hasToken = false
	}

	for _, r := range s {
		switch {
		case escape:
			token.WriteRune(r)
			escape = false
			hasToken = true
		case inSingle:
			if r == '\'' {
				inSingle = false
			} else {
				token.WriteRune(r)
			}
			hasToken = true
		case inDouble:
			if r == '"' {
				inDouble = false
			} else if r == '\\' {
				escape = true
			} else {
				token.WriteRune(r)
			}
			hasToken = true
		default:
			if unicode.IsSpace(r) {
				flush()
				continue
			}
			switch r {
			case '\\':
				escape = true
				hasToken = true
			case '\'':
				inSingle = true
				hasToken = true
			case '"':
				inDouble = true
				hasToken = true
			default:
				token.WriteRune(r)
				hasToken = true
			}
		}
	}

	if escape || inSingle || inDouble {
		return nil, false
	}
	flush()
	return words, true
}

func isTerminalApp(app string) bool {
	a := strings.ToLower(strings.TrimSpace(app))
	return a == "terminal" || a == "terminal.app"
}

func isSystemDefaultTerminal(app string) bool {
	a := strings.ToLower(strings.TrimSpace(app))
	return a == "default" || a == "system default" || a == "macos default"
}

func isGhosttyApp(app string) bool {
	return strings.Contains(strings.ToLower(app), "ghostty")
}

func shQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func formatCommandArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func writeDefaultTerminalCommandScript(binary, mountPath string) (string, error) {
	f, err := os.CreateTemp("", "cardbot-launch-*.command")
	if err != nil {
		return "", fmt.Errorf("creating command script: %w", err)
	}
	defer f.Close()

	scriptPath := f.Name()
	script := fmt.Sprintf("#!/bin/sh\nrm -- %s\nexec %s %s\n", shQuote(scriptPath), shQuote(binary), shQuote(mountPath))
	if _, err := f.WriteString(script); err != nil {
		_ = os.Remove(scriptPath)
		return "", fmt.Errorf("writing command script: %w", err)
	}
	if err := f.Chmod(0o700); err != nil {
		_ = os.Remove(scriptPath)
		return "", fmt.Errorf("chmod command script: %w", err)
	}
	return filepath.Clean(scriptPath), nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
		}
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
	}
	return nil
}
