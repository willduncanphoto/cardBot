package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/illwill/cardbot/internal/config"
)

// SetupPrompter reads setup answers from a shared buffered input stream.
// Reusing one reader prevents buffered read-ahead from consuming subsequent answers.
type SetupPrompter struct {
	reader *bufio.Reader
	out    io.Writer
}

// NewSetupPrompter creates a setup prompter with shared input/output.
func NewSetupPrompter(in io.Reader, out io.Writer) *SetupPrompter {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return &SetupPrompter{reader: bufio.NewReader(in), out: out}
}

func (p *SetupPrompter) PromptNamingMode(defaultMode string) string {
	return promptNamingModeReader(p.reader, p.out, defaultMode)
}

func (p *SetupPrompter) PromptDaemonEnabled(defaultEnabled bool) bool {
	return promptDaemonEnabledReader(p.reader, p.out, defaultEnabled)
}

func (p *SetupPrompter) PromptDaemonStartAtLogin(defaultEnabled bool) bool {
	return promptDaemonStartAtLoginReader(p.reader, p.out, defaultEnabled)
}

func (p *SetupPrompter) PromptDaemonTerminalApp(defaultApp string) string {
	return promptDaemonTerminalAppReader(p.reader, p.out, defaultApp)
}

func (p *SetupPrompter) PromptDaemonWorkingDirectory(defaultDir string) string {
	return promptDaemonWorkingDirectoryReader(p.reader, p.out, defaultDir)
}

// RunSetup executes first-time/--setup prompts and persists config.
func RunSetup(
	cfg *config.Config,
	cfgPath string,
	promptDestinationFn func(string) string,
	promptNamingFn func(string) string,
	promptDaemonEnabledFn func(bool) bool,
	promptDaemonStartAtLoginFn func(bool) bool,
	promptDaemonTerminalAppFn func(string) string,
	promptDaemonWorkingDirectoryFn func(string) string,
) error {
	cfg.Destination.Path = config.ContractPath(promptDestinationFn(cfg.Destination.Path))
	cfg.Naming.Mode = config.NormalizeNamingMode(promptNamingFn(cfg.Naming.Mode))
	cfg.Daemon.Enabled = promptDaemonEnabledFn(cfg.Daemon.Enabled)
	if cfg.Daemon.Enabled {
		cfg.Daemon.StartAtLogin = promptDaemonStartAtLoginFn(cfg.Daemon.StartAtLogin)
		cfg.Daemon.TerminalApp = normalizeDaemonTerminalApp(promptDaemonTerminalAppFn(cfg.Daemon.TerminalApp))
		if promptDaemonWorkingDirectoryFn != nil {
			cfg.Daemon.WorkingDirectory = normalizeDaemonWorkingDirectory(promptDaemonWorkingDirectoryFn(cfg.Daemon.WorkingDirectory))
		} else {
			cfg.Daemon.WorkingDirectory = normalizeDaemonWorkingDirectory(cfg.Daemon.WorkingDirectory)
		}
	} else {
		cfg.Daemon.StartAtLogin = false
	}
	cfg.Daemon.WorkingDirectory = normalizeDaemonWorkingDirectory(cfg.Daemon.WorkingDirectory)

	if cfgPath == "" {
		return nil
	}
	return config.Save(cfg, cfgPath)
}

// PromptNamingMode asks the user how filenames should be written on copy.
func PromptNamingMode(defaultMode string) string {
	return promptNamingModeIO(os.Stdin, os.Stdout, defaultMode)
}

// PromptDaemonEnabled asks whether CardBot should auto-launch from daemon mode.
func PromptDaemonEnabled(defaultEnabled bool) bool {
	return promptDaemonEnabledIO(os.Stdin, os.Stdout, defaultEnabled)
}

// PromptDaemonStartAtLogin asks whether daemon mode should auto-start at login.
func PromptDaemonStartAtLogin(defaultEnabled bool) bool {
	return promptDaemonStartAtLoginIO(os.Stdin, os.Stdout, defaultEnabled)
}

// PromptDaemonTerminalApp asks which terminal app daemon mode should launch.
func PromptDaemonTerminalApp(defaultApp string) string {
	return promptDaemonTerminalAppIO(os.Stdin, os.Stdout, defaultApp)
}

// PromptDaemonWorkingDirectory asks which working directory daemon-launched
// terminal windows should start in.
func PromptDaemonWorkingDirectory(defaultDir string) string {
	return promptDaemonWorkingDirectoryIO(os.Stdin, os.Stdout, defaultDir)
}

func promptNamingModeIO(in io.Reader, out io.Writer, defaultMode string) string {
	return promptNamingModeReader(bufio.NewReader(in), out, defaultMode)
}

func promptNamingModeReader(reader *bufio.Reader, out io.Writer, defaultMode string) string {
	mode := config.NormalizeNamingMode(defaultMode)

	for {
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out, "File Naming")
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Camera filenames reset every 10,000 shots.")
		fmt.Fprintln(out, "This can cause duplicates when copying multiple cards.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "[1] Keep camera filenames")
		fmt.Fprintln(out, "    DSC_0001.NEF, DSC_0002.NEF ...")
		fmt.Fprintln(out, "    Use if you rely on camera numbering.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "[2] Timestamp + sequence")
		fmt.Fprintln(out, "    260314T143052_0001.NEF, _0002.NEF ...")
		fmt.Fprintln(out, "    Use for automatic order across all cards.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "You can change this later with cardbot --setup.")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Choice [%s]: ", namingChoiceDefault(mode))

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out)
			return mode
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(out, "Naming set to: %s\n", namingModeLabel(mode))
			return mode
		}

		if chosen, ok := parseNamingChoice(line); ok {
			fmt.Fprintf(out, "Naming set to: %s\n", namingModeLabel(chosen))
			return chosen
		}

		fmt.Fprintln(out, "Please enter 1 or 2.")
		fmt.Fprintln(out)
	}
}

func promptDaemonEnabledIO(in io.Reader, out io.Writer, defaultEnabled bool) bool {
	return promptDaemonEnabledReader(bufio.NewReader(in), out, defaultEnabled)
}

func promptDaemonEnabledReader(reader *bufio.Reader, out io.Writer, defaultEnabled bool) bool {
	defaultChoice := "n"
	if defaultEnabled {
		defaultChoice = "y"
	}

	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Background Auto-Launch")
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out, "When daemon mode is running, launch CardBot")
		fmt.Fprintln(out, "automatically when a memory card is connected? [y/n]")
		fmt.Fprintf(out, "Choice [%s]: ", defaultChoice)

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out)
			return defaultEnabled
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(out, "Background auto-launch: %s\n", enabledLabel(defaultEnabled))
			return defaultEnabled
		}

		if enabled, ok := parseYesNo(line); ok {
			fmt.Fprintf(out, "Background auto-launch: %s\n", enabledLabel(enabled))
			return enabled
		}

		fmt.Fprintln(out, "Please enter y or n.")
	}
}

func promptDaemonStartAtLoginIO(in io.Reader, out io.Writer, defaultEnabled bool) bool {
	return promptDaemonStartAtLoginReader(bufio.NewReader(in), out, defaultEnabled)
}

func promptDaemonStartAtLoginReader(reader *bufio.Reader, out io.Writer, defaultEnabled bool) bool {
	defaultChoice := "n"
	if defaultEnabled {
		defaultChoice = "y"
	}

	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Start at Login")
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out, "Start CardBot daemon automatically")
		fmt.Fprintln(out, "when you log in? [y/n]")
		fmt.Fprintf(out, "Choice [%s]: ", defaultChoice)

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out)
			return defaultEnabled
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(out, "Start-at-login: %s\n", enabledLabel(defaultEnabled))
			return defaultEnabled
		}

		if enabled, ok := parseYesNo(line); ok {
			fmt.Fprintf(out, "Start-at-login: %s\n", enabledLabel(enabled))
			return enabled
		}

		fmt.Fprintln(out, "Please enter y or n.")
	}
}

func promptDaemonTerminalAppIO(in io.Reader, out io.Writer, defaultApp string) string {
	return promptDaemonTerminalAppReader(bufio.NewReader(in), out, defaultApp)
}

func promptDaemonTerminalAppReader(reader *bufio.Reader, out io.Writer, defaultApp string) string {
	app := normalizeDaemonTerminalApp(defaultApp)

	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Daemon Terminal App")
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out, "Choose which terminal app to open when")
		fmt.Fprintln(out, "a card is detected in daemon mode:")
		fmt.Fprintln(out, "[1] Use macOS default terminal app")
		fmt.Fprintln(out, "[2] Terminal")
		fmt.Fprintln(out, "[3] Ghostty")
		fmt.Fprintln(out, "[4] Custom app name")
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Choice [%s]: ", daemonTerminalDefaultChoice(app))

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Daemon terminal app: %s\n", app)
			return app
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(out, "Daemon terminal app: %s\n", app)
			return app
		}

		chosen, ok := parseDaemonTerminalChoice(line)
		if !ok {
			fmt.Fprintln(out, "Please enter 1, 2, 3, or 4.")
			continue
		}
		if chosen == "" {
			fmt.Fprintf(out, "App name: ")
			customLine, customErr := reader.ReadString('\n')
			if customErr != nil {
				fmt.Fprintln(out)
				fmt.Fprintf(out, "Daemon terminal app: %s\n", app)
				return app
			}
			custom := strings.TrimSpace(customLine)
			if custom == "" {
				fmt.Fprintln(out, "Please enter a non-empty app name.")
				continue
			}
			chosen = custom
		}

		chosen = normalizeDaemonTerminalApp(chosen)
		fmt.Fprintf(out, "Daemon terminal app: %s\n", chosen)
		return chosen
	}
}

func promptDaemonWorkingDirectoryIO(in io.Reader, out io.Writer, defaultDir string) string {
	return promptDaemonWorkingDirectoryReader(bufio.NewReader(in), out, defaultDir)
}

func promptDaemonWorkingDirectoryReader(reader *bufio.Reader, out io.Writer, defaultDir string) string {
	def := normalizeDaemonWorkingDirectory(defaultDir)

	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Daemon Window Working Directory")
		fmt.Fprintln(out, "────────────────────────────────────────")
		fmt.Fprintln(out, "Where should daemon-launched terminal windows start?")
		fmt.Fprintln(out, "Use ~ for your home folder or enter any path.")
		fmt.Fprintf(out, "Path [%s]: ", def)

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Daemon working directory: %s\n", def)
			return def
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Fprintf(out, "Daemon working directory: %s\n", def)
			return def
		}

		chosen := normalizeDaemonWorkingDirectory(line)
		fmt.Fprintf(out, "Daemon working directory: %s\n", chosen)
		return chosen
	}
}

func parseDaemonTerminalChoice(input string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "1", "default", "system default", "macos default":
		return "Default", true
	case "2", "terminal", "terminal.app":
		return "Terminal", true
	case "3", "ghostty":
		return "Ghostty", true
	case "4", "custom":
		return "", true
	default:
		return "", false
	}
}

func daemonTerminalDefaultChoice(app string) string {
	switch strings.ToLower(strings.TrimSpace(app)) {
	case "default", "system default", "macos default":
		return "1"
	case "terminal", "terminal.app":
		return "2"
	case "ghostty":
		return "3"
	default:
		return "4"
	}
}

func normalizeDaemonTerminalApp(app string) string {
	app = strings.TrimSpace(app)
	if app == "" {
		return "Terminal"
	}
	if strings.EqualFold(app, "default") || strings.EqualFold(app, "system default") || strings.EqualFold(app, "macos default") {
		return "Default"
	}
	if strings.EqualFold(app, "terminal.app") {
		return "Terminal"
	}
	if strings.EqualFold(app, "ghostty") {
		return "Ghostty"
	}
	return app
}

func normalizeDaemonWorkingDirectory(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "~"
	}
	if strings.EqualFold(dir, "home") {
		return "~"
	}
	return config.ContractPath(dir)
}

func parseYesNo(input string) (bool, bool) {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "y", "yes":
		return true, true
	case "n", "no":
		return false, true
	default:
		return false, false
	}
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func parseNamingChoice(input string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "1", "o", "original":
		return config.NamingOriginal, true
	case "2", "t", "timestamp":
		return config.NamingTimestamp, true
	default:
		return "", false
	}
}

func namingChoiceDefault(mode string) string {
	if config.NormalizeNamingMode(mode) == config.NamingTimestamp {
		return "2"
	}
	return "1"
}

func namingModeLabel(mode string) string {
	if config.NormalizeNamingMode(mode) == config.NamingTimestamp {
		return "Timestamp + sequence"
	}
	return "Camera original"
}

func namingDisplayLine(mode string) string {
	if config.NormalizeNamingMode(mode) != config.NamingTimestamp {
		return "Camera original"
	}
	return "Timestamp + sequence"
}
