//go:build !darwin

package speedtest

import "errors"

// Result holds the measured read and write speeds.
type Result struct {
	WriteSpeed float64
	ReadSpeed  float64
}

// Run is not supported on this platform.
func Run(mountPath string, onProgress func(phase string, mbps float64)) (*Result, error) {
	return nil, errors.New("speed test not supported on this platform")
}
