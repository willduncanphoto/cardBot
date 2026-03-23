package main

import (
	"fmt"
	"os"
	"strings"
)

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

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func readRecentLauncherExecLines(logPath string, limit int) ([]string, error) {
	if strings.TrimSpace(logPath) == "" {
		return nil, fmt.Errorf("log path is empty")
	}
	if limit <= 0 {
		return []string{}, nil
	}

	lines, err := readRecentMatchingLogLines(logPath, "Launcher exec:", limit)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		lines = []string{}
	}

	if len(lines) < limit {
		remaining := limit - len(lines)
		older, oldErr := readRecentMatchingLogLines(logPath+".old", "Launcher exec:", remaining)
		if oldErr == nil {
			lines = append(lines, older...)
		} else if !os.IsNotExist(oldErr) {
			return nil, oldErr
		}
	}

	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, nil
}

func readRecentMatchingLogLines(path, needle string, limit int) ([]string, error) {
	if limit <= 0 {
		return []string{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawLines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	matches := make([]string, 0, limit)
	for i := len(rawLines) - 1; i >= 0 && len(matches) < limit; i-- {
		line := strings.TrimSpace(rawLines[i])
		if line == "" {
			continue
		}
		if strings.Contains(line, needle) {
			matches = append(matches, line)
		}
	}
	return matches, nil
}
