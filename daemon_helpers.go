package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/illwill/cardbot/app"
	"github.com/illwill/cardbot/config"
	"github.com/illwill/cardbot/launchagent"
)

func normalizeDaemonTerminalAppForLaunch(_ string) string {
	// CardBot daemon launches use Terminal.app via AppleScript.
	// This avoids the ugly .command script header that "Default" produces.
	return "Terminal"
}

func daemonTerminalAppLabel(name string) string {
	if strings.EqualFold(strings.TrimSpace(name), "default") {
		return "Default (macOS)"
	}
	return name
}

func resolveDaemonWorkingDirectory(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "~"
	}
	expanded, err := config.ExpandPath(raw)
	if err != nil || strings.TrimSpace(expanded) == "" {
		if home, homeErr := os.UserHomeDir(); homeErr == nil && strings.TrimSpace(home) != "" {
			return home
		}
		return raw
	}
	return expanded
}

func daemonWorkingDirectoryLabel(configValue, resolved string) string {
	if strings.TrimSpace(configValue) == "" {
		configValue = "~"
	}
	if strings.TrimSpace(resolved) == "" {
		return configValue
	}
	if config.ContractPath(resolved) == configValue {
		return configValue
	}
	return fmt.Sprintf("%s (%s)", configValue, resolved)
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

func syncDaemonAutoStartFromConfig(cfg *config.Config) {
	if runtime.GOOS != "darwin" {
		return
	}

	if cfg == nil {
		return
	}

	if cfg.Daemon.Enabled && cfg.Daemon.StartAtLogin {
		// Always (re)install from setup to keep LaunchAgent ProgramArguments
		// aligned with the current cardbot binary path and avoid stale wrappers.
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

	if st, err := launchagent.CurrentStatus(); err == nil && !st.Installed {
		fmt.Printf("[%s] Start-at-login already disabled\n", app.Ts())
		return
	}

	if _, err := launchagent.Uninstall(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not uninstall launch agent: %v\n", err)
		return
	}
	fmt.Printf("[%s] Start-at-login disabled\n", app.Ts())
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
