package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/illwill/cardbot/internal/app"
	"github.com/illwill/cardbot/internal/config"
	"github.com/illwill/cardbot/internal/daemon"
	"github.com/illwill/cardbot/internal/instance"
	"github.com/illwill/cardbot/internal/launchagent"
	"github.com/illwill/cardbot/internal/launcher"
	cblog "github.com/illwill/cardbot/internal/log"
	"github.com/illwill/cardbot/internal/pick"
)

const version = "0.5.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "self-update":
			os.Exit(app.RunSelfUpdate(version))
		case "install-daemon":
			os.Exit(runInstallDaemonCommand())
		case "uninstall-daemon":
			os.Exit(runUninstallDaemonCommand())
		case "daemon-status":
			os.Exit(runDaemonStatusCommand(os.Args[2:]))
		}
	}

	// --- CLI flags ---
	var (
		flagVersion = flag.Bool("version", false, "print version and exit")
		flagDest    = flag.String("dest", "", "destination path for copied cards")
		flagDryRun  = flag.Bool("dry-run", false, "scan cards but do not copy files")
		flagReset   = flag.Bool("reset", false, "clear saved config and exit")
		flagSetup   = flag.Bool("setup", false, "re-run first-time setup (destination, naming, daemon options)")
		flagDaemon  = flag.Bool("daemon", false, "run as background daemon watching for cards")
	)
	flag.Parse()

	if *flagVersion {
		fmt.Printf("cardbot %s\n", version)
		os.Exit(0)
	}

	if *flagReset {
		cfgPath, err := config.Path()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not determine config path: %v\n", err)
			os.Exit(1)
		}
		if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: could not remove config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config cleared. Please restart CardBot.")
		os.Exit(0)
	}

	// --- Load config ---
	cfgPath, err := config.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine config path: %v\n", err)
		cfgPath = ""
	}

	var cfg *config.Config
	var cfgWarnings []string

	if cfgPath != "" {
		cfg, cfgWarnings, err = config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %s — using defaults\n", app.FriendlyErr(err))
			cfg = config.Defaults()
		}
	} else {
		cfg = config.Defaults()
	}

	// --- CLI flags override config ---
	if *flagDest != "" {
		cfg.Destination.Path = *flagDest
	}

	// --- First-run or --setup: prompt for destination, then continue into the app ---
	needsSetup := *flagSetup
	if cfgPath != "" {
		if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
			needsSetup = true
		}
	}
	if needsSetup {
		setupReader := bufio.NewReader(os.Stdin)
		setupPrompter := app.NewSetupPrompter(setupReader, os.Stdout)
		promptDestinationFn := func(defaultPath string) string {
			return promptDestinationWithIO(defaultPath, setupReader, os.Stdout)
		}
		if saveErr := app.RunSetup(cfg, cfgPath, promptDestinationFn, setupPrompter.PromptNamingMode, setupPrompter.PromptDaemonEnabled, setupPrompter.PromptDaemonStartAtLogin, setupPrompter.PromptDaemonTerminalApp); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save config: %s\n", app.FriendlyErr(saveErr))
		}
		syncDaemonAutoStartFromConfig(cfg)
		printSetupSummary(cfg)
		fmt.Println()
	}

	// --- Set up logger ---
	var logger *cblog.Logger
	if cfg.Advanced.LogFile != "" {
		logPath, expandErr := config.ExpandPath(cfg.Advanced.LogFile)
		if expandErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not expand log path: %s\n", app.FriendlyErr(expandErr))
		} else {
			logger, err = cblog.Open(logPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not open log file: %s\n", app.FriendlyErr(err))
			} else {
				defer logger.Close()
			}
		}
	}

	// --- Daemon mode ---
	if *flagDaemon {
		runDaemon(cfg, logger)
		return
	}

	// --- Build app ---
	targetPath := ""
	if args := flag.Args(); len(args) > 0 {
		targetPath = args[0]
	}

	a := app.New(app.Config{
		Cfg:        cfg,
		Logger:     logger,
		DryRun:     *flagDryRun,
		Version:    version,
		TargetPath: targetPath,
	})

	// Print any config warnings now that logging is ready.
	for _, w := range cfgWarnings {
		a.Printf("[%s] Warning: %s\n", app.Ts(), w)
	}

	a.Printf("[%s] CardBot %s\n", app.Ts(), version)
	if cfg.Daemon.Enabled {
		a.Printf("[%s] Auto-launch preference is enabled (run daemon mode to watch in background)\n", app.Ts())
	}
	if cfg.Daemon.StartAtLogin {
		a.Printf("[%s] Start-at-login is enabled\n", app.Ts())
	}

	// Startup update-check with spinner, then update the same line with result.
	ts := app.Ts()
	prefix := fmt.Sprintf("[%s] ", ts)
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = prefix + "Checking for updates "
	s.Start()
	latest, updateErr := app.MaybeCheckForUpdate(logger, version)
	time.Sleep(500 * time.Millisecond) // ensure user sees activity
	s.Stop()

	// Update the same line with the result.
	clearEOL := "\033[K"
	var msg string
	if updateErr != nil {
		msg = "NO SIGNAL"
	} else if latest != "" {
		msg = fmt.Sprintf("UPDATE AVAILABLE (%s)", latest)
	} else {
		msg = "CardBot is up to date"
	}
	fmt.Printf("\r%s%s%s\n", prefix, msg, clearEOL)

	// Print action line if update available.
	if latest != "" && updateErr == nil {
		fmt.Printf("%sRun: cardbot self-update\n", prefix)
	}

	if *flagDryRun {
		a.Printf("[%s] Dry-run mode — no files will be copied\n", app.Ts())
	}

	if targetPath == "" {
		a.StartScanning()
	}

	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// promptDestination asks the user to pick a destination path.
// On macOS, opens the native folder picker. Falls back to readline on Linux.
func promptDestination(defaultPath string) string {
	return promptDestinationWithIO(defaultPath, bufio.NewReader(os.Stdin), os.Stdout)
}

func promptDestinationWithIO(defaultPath string, in *bufio.Reader, out io.Writer) string {
	if in == nil {
		in = bufio.NewReader(os.Stdin)
	}
	if out == nil {
		out = os.Stdout
	}

	fmt.Fprintln(out, "Welcome to CardBot!")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Where should CardBot copy your work?")
	fmt.Fprintln(out)

	expanded, err := config.ExpandPath(defaultPath)
	if err != nil {
		expanded = defaultPath
	}

	picked, err := pick.Folder(expanded)
	if err == nil && picked != "" {
		fmt.Fprintf(out, "Destination: %s\n", picked)
		return picked
	}

	// Fallback: readline with shared buffered input.
	return promptDestinationReadlineIO(expanded, in, out)
}

// promptDestinationReadline is the fallback path prompt using stdlib.
func promptDestinationReadline(defaultPath string) string {
	return promptDestinationReadlineIO(defaultPath, bufio.NewReader(os.Stdin), os.Stdout)
}

func promptDestinationReadlineIO(defaultPath string, in *bufio.Reader, out io.Writer) string {
	if in == nil {
		in = bufio.NewReader(os.Stdin)
	}
	if out == nil {
		out = os.Stdout
	}

	fmt.Fprintf(out, "Destination [%s]: ", defaultPath)
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultPath
	}
	return line
}

// runDaemon starts the background daemon that watches for card insertions
// and launches a terminal window with cardbot targeting the detected card.
func runDaemon(cfg *config.Config, logger *cblog.Logger) {
	if cfg == nil {
		cfg = config.Defaults()
	}

	cardbotBinary, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine executable path: %v\n", err)
		os.Exit(1)
	}

	logf := func(format string, args ...any) {
		if logger != nil {
			logger.Printf(format, args...)
		}
	}

	appName := strings.TrimSpace(cfg.Daemon.TerminalApp)
	if appName == "" {
		appName = "Terminal"
	}
	fmt.Printf("[%s] Daemon terminal app: %s\n", time.Now().Format("2006-01-02T15:04:05"), appName)
	if len(cfg.Daemon.LaunchArgs) > 0 {
		fmt.Printf("[%s] Daemon custom launch args enabled\n", time.Now().Format("2006-01-02T15:04:05"))
	}
	processName := filepath.Base(cardbotBinary)

	d := daemon.New(daemon.Config{
		OnCardInserted: func(path string) {
			hasOther, checkErr := instance.HasOtherProcess(processName, os.Getpid())
			if checkErr != nil {
				fmt.Fprintf(os.Stderr, "[%s] Warning: single-instance check failed (%v)\n", time.Now().Format("2006-01-02T15:04:05"), checkErr)
				logf("Single-instance check failed: %v", checkErr)
			} else if hasOther {
				fmt.Printf("[%s] CardBot already running in another process — skipping auto-launch\n", time.Now().Format("2006-01-02T15:04:05"))
				logf("Auto-launch skipped for %s: another cardbot process is running", path)
				return
			}

			err := launcher.Launch(launcher.Options{
				TerminalApp:   appName,
				LaunchArgs:    cfg.Daemon.LaunchArgs,
				CardBotBinary: cardbotBinary,
				MountPath:     path,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "[%s] Launch failed: %v\n", time.Now().Format("2006-01-02T15:04:05"), err)
				if hint := daemonLaunchHint(err); hint != "" {
					fmt.Fprintf(os.Stderr, "[%s] Hint: %s\n", time.Now().Format("2006-01-02T15:04:05"), hint)
					logf("Launch hint for %s: %s", path, hint)
				}
				logf("Launch failed for %s: %v", path, err)
				return
			}
			fmt.Printf("[%s] Launched %s for %s\n", time.Now().Format("2006-01-02T15:04:05"), appName, path)
			logf("Launched terminal app %q for %s", appName, path)
		},
	})

	if err := d.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func daemonLaunchHint(err error) string {
	if err == nil {
		return ""
	}
	s := strings.ToLower(err.Error())

	automationMarkers := []string{
		"not authorized to send apple events",
		"not authorised to send apple events",
		"automation",
		"-1743",
		"erraeeventnotpermitted",
	}
	if containsAny(s, automationMarkers...) {
		return "Grant Automation permission in macOS System Settings → Privacy & Security → Automation for your terminal app."
	}

	fullDiskAccessMarkers := []string{
		"operation not permitted",
		"permission denied",
		" eperm",
	}
	if containsAny(s, fullDiskAccessMarkers...) {
		return "Grant Full Disk Access to CardBot and your terminal app in macOS System Settings → Privacy & Security → Full Disk Access."
	}
	return ""
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func runInstallDaemonCommand() int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine executable path: %v\n", err)
		return 1
	}

	plist, err := launchagent.Install(exe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	updateSavedDaemonPrefs(func(cfg *config.Config) {
		cfg.Daemon.Enabled = true
		cfg.Daemon.StartAtLogin = true
	})

	fmt.Printf("Installed LaunchAgent: %s\n", plist)
	fmt.Println("CardBot daemon will start at login.")
	fmt.Println("Tip: run `cardbot --setup` anytime to change daemon/login preferences.")
	return 0
}

func runUninstallDaemonCommand() int {
	plist, err := launchagent.Uninstall()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	updateSavedDaemonPrefs(func(cfg *config.Config) {
		cfg.Daemon.StartAtLogin = false
	})

	fmt.Printf("Uninstalled LaunchAgent: %s\n", plist)
	fmt.Println("CardBot daemon will no longer start at login.")
	fmt.Println("Tip: run `cardbot --setup` anytime to change daemon/login preferences.")
	return 0
}

type daemonStatusOptions struct {
	JSON bool
}

type daemonStatusReport struct {
	Version             string                    `json:"version"`
	PID                 int                       `json:"pid"`
	ConfigPath          string                    `json:"config_path,omitempty"`
	ConfigPathError     string                    `json:"config_path_error,omitempty"`
	ConfigLoadError     string                    `json:"config_load_error,omitempty"`
	ConfigWarnings      []string                  `json:"config_warnings,omitempty"`
	Daemon              daemonStatusDaemonReport  `json:"daemon"`
	SingleInstanceGuard daemonStatusSIGuardReport `json:"single_instance_guard"`
	LaunchAgent         daemonStatusLAReport      `json:"launch_agent"`
}

type daemonStatusDaemonReport struct {
	Enabled      bool     `json:"enabled"`
	StartAtLogin bool     `json:"start_at_login"`
	TerminalApp  string   `json:"terminal_app"`
	LaunchArgs   []string `json:"launch_args"`
}

type daemonStatusSIGuardReport struct {
	Enabled         bool   `json:"enabled"`
	ProcessName     string `json:"process_name"`
	HasOtherProcess bool   `json:"has_other_process"`
	CheckError      string `json:"check_error,omitempty"`
}

type daemonStatusLAReport struct {
	Supported bool   `json:"supported"`
	PlistPath string `json:"plist_path,omitempty"`
	Installed bool   `json:"installed"`
	Loaded    bool   `json:"loaded"`
	Error     string `json:"error,omitempty"`
}

func runDaemonStatusCommand(args []string) int {
	opts, err := parseDaemonStatusOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	report := collectDaemonStatusReport()
	if opts.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not encode daemon status JSON: %v\n", err)
			return 1
		}
		return 0
	}

	printDaemonStatusReport(report)
	return 0
}

func parseDaemonStatusOptions(args []string) (daemonStatusOptions, error) {
	fs := flag.NewFlagSet("daemon-status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOut := fs.Bool("json", false, "output daemon status as JSON")
	if err := fs.Parse(args); err != nil {
		return daemonStatusOptions{}, err
	}
	if fs.NArg() > 0 {
		return daemonStatusOptions{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	return daemonStatusOptions{JSON: *jsonOut}, nil
}

func collectDaemonStatusReport() daemonStatusReport {
	processName := "cardbot"
	if exe, err := os.Executable(); err == nil {
		processName = filepath.Base(exe)
	}
	pid := os.Getpid()
	report := daemonStatusReport{
		Version:             version,
		PID:                 pid,
		SingleInstanceGuard: collectSingleInstanceGuardStatus(processName, pid, instance.HasOtherProcess),
		LaunchAgent:         daemonStatusLAReport{Supported: runtime.GOOS == "darwin"},
	}

	cfg := config.Defaults()
	cfgPath, cfgPathErr := config.Path()
	if cfgPathErr != nil {
		report.ConfigPathError = cfgPathErr.Error()
	} else {
		report.ConfigPath = cfgPath
		loaded, warnings, loadErr := config.Load(cfgPath)
		if loadErr != nil {
			report.ConfigLoadError = loadErr.Error()
		} else {
			cfg = loaded
			report.ConfigWarnings = warnings
		}
	}

	launchArgs := cfg.Daemon.LaunchArgs
	if launchArgs == nil {
		launchArgs = []string{}
	}
	report.Daemon = daemonStatusDaemonReport{
		Enabled:      cfg.Daemon.Enabled,
		StartAtLogin: cfg.Daemon.StartAtLogin,
		TerminalApp:  cfg.Daemon.TerminalApp,
		LaunchArgs:   launchArgs,
	}

	if !report.LaunchAgent.Supported {
		return report
	}

	st, err := launchagent.CurrentStatus()
	if err != nil {
		report.LaunchAgent.Error = err.Error()
		return report
	}
	report.LaunchAgent.PlistPath = st.PlistPath
	report.LaunchAgent.Installed = st.Installed
	report.LaunchAgent.Loaded = st.Loaded
	return report
}

func collectSingleInstanceGuardStatus(processName string, selfPID int, checker func(processName string, selfPID int) (bool, error)) daemonStatusSIGuardReport {
	report := daemonStatusSIGuardReport{
		Enabled:     true,
		ProcessName: processName,
	}
	if checker == nil {
		report.CheckError = "no checker configured"
		return report
	}
	hasOther, err := checker(processName, selfPID)
	if err != nil {
		report.CheckError = err.Error()
		return report
	}
	report.HasOtherProcess = hasOther
	return report
}

func printDaemonStatusReport(report daemonStatusReport) {
	fmt.Println("CardBot Daemon Status")
	fmt.Println("────────────────────────────────────────")
	fmt.Printf("Version: %s\n", report.Version)
	fmt.Printf("PID: %d\n", report.PID)

	if report.ConfigPathError != "" {
		fmt.Printf("Config path: unavailable (%s)\n", report.ConfigPathError)
	} else {
		fmt.Printf("Config path: %s\n", report.ConfigPath)
	}
	if report.ConfigLoadError != "" {
		fmt.Printf("Config load: error (%s), using defaults\n", report.ConfigLoadError)
	}
	if len(report.ConfigWarnings) > 0 {
		fmt.Printf("Config warnings: %d\n", len(report.ConfigWarnings))
	}

	fmt.Printf("Daemon enabled: %s\n", boolEnabled(report.Daemon.Enabled))
	fmt.Printf("Start at login: %s\n", boolEnabled(report.Daemon.StartAtLogin))
	fmt.Printf("Terminal app: %s\n", report.Daemon.TerminalApp)
	if len(report.Daemon.LaunchArgs) == 0 {
		fmt.Println("Launch args: (default)")
	} else {
		fmt.Printf("Launch args: %v\n", report.Daemon.LaunchArgs)
	}

	fmt.Printf("Single-instance guard: %s\n", boolEnabled(report.SingleInstanceGuard.Enabled))
	fmt.Printf("Guard process name: %s\n", report.SingleInstanceGuard.ProcessName)
	if report.SingleInstanceGuard.CheckError != "" {
		fmt.Printf("Guard check: error (%s)\n", report.SingleInstanceGuard.CheckError)
	} else {
		fmt.Printf("Other CardBot process running: %s\n", boolYesNo(report.SingleInstanceGuard.HasOtherProcess))
	}

	if !report.LaunchAgent.Supported {
		fmt.Println("LaunchAgent: unsupported on this platform")
		return
	}
	if report.LaunchAgent.Error != "" {
		fmt.Printf("LaunchAgent status: error (%s)\n", report.LaunchAgent.Error)
		return
	}

	fmt.Printf("LaunchAgent plist: %s\n", report.LaunchAgent.PlistPath)
	fmt.Printf("LaunchAgent installed: %s\n", boolEnabled(report.LaunchAgent.Installed))
	fmt.Printf("LaunchAgent loaded: %s\n", boolEnabled(report.LaunchAgent.Loaded))
}

func boolEnabled(v bool) string {
	if v {
		return "enabled"
	}
	return "disabled"
}

func boolYesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func syncDaemonAutoStartFromConfig(cfg *config.Config) {
	if runtime.GOOS != "darwin" {
		return
	}

	if cfg == nil {
		return
	}

	if cfg.Daemon.Enabled && cfg.Daemon.StartAtLogin {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not determine executable path for launch agent install: %v\n", err)
			return
		}
		if _, err := launchagent.Install(exe); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not install launch agent: %v\n", err)
			return
		}
		fmt.Printf("[%s] Start-at-login enabled\n", app.Ts())
		return
	}

	if _, err := launchagent.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not uninstall launch agent: %v\n", err)
		return
	}
	fmt.Printf("[%s] Start-at-login disabled\n", app.Ts())
}

func printSetupSummary(cfg *config.Config) {
	if cfg == nil {
		return
	}

	fmt.Println("Setup saved.")
	if cfg.Daemon.Enabled {
		fmt.Println("- Background auto-launch: enabled")
		fmt.Printf("- Daemon terminal app: %s\n", cfg.Daemon.TerminalApp)
	} else {
		fmt.Println("- Background auto-launch: disabled")
	}

	if cfg.Daemon.StartAtLogin {
		fmt.Println("- Start daemon at login: enabled")
	} else {
		fmt.Println("- Start daemon at login: disabled")
	}

	if cfg.Daemon.Enabled && !cfg.Daemon.StartAtLogin {
		fmt.Println("Tip: start background mode now with `cardbot --daemon`.")
	}
	fmt.Println("Tip: run `cardbot --setup` anytime to change these settings.")
}

func updateSavedDaemonPrefs(mutator func(cfg *config.Config)) {
	if mutator == nil {
		return
	}

	cfgPath, err := config.Path()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not determine config path to update daemon preferences: %v\n", err)
		return
	}

	cfg, _, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config to update daemon preferences: %v\n", err)
		cfg = config.Defaults()
	}
	if cfg == nil {
		cfg = config.Defaults()
	}

	mutator(cfg)
	if !cfg.Daemon.Enabled {
		cfg.Daemon.StartAtLogin = false
	}
	if strings.TrimSpace(cfg.Daemon.TerminalApp) == "" {
		cfg.Daemon.TerminalApp = "Terminal"
	}

	if err := config.Save(cfg, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save config daemon preferences: %v\n", err)
	}
}
