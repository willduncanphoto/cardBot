//go:build !darwin && !linux

package detect

import (
	"fmt"
)

// Detector monitors for memory card insertion/removal.
type Detector struct {
	cards    map[string]*Card
	events   chan *Card
	removals chan string
}

// NewDetector creates a new card detector.
func NewDetector() *Detector {
	return &Detector{
		cards:    make(map[string]*Card),
		events:   make(chan *Card, 10),
		removals: make(chan string, 10),
	}
}

// Start begins monitoring for card insertion/removal.
func (d *Detector) Start() error {
	return fmt.Errorf("memory card detection not supported on this platform")
}

// Stop halts card monitoring.
func (d *Detector) Stop() {}

// Events returns a channel for card insertion events.
func (d *Detector) Events() <-chan *Card {
	return d.events
}

// Removals returns a channel for card removal events.
func (d *Detector) Removals() <-chan string {
	return d.removals
}

// Remove is a no-op on unsupported platforms.
func (d *Detector) Remove(path string) {}

// Eject is not supported on this platform.
func (d *Detector) Eject(path string) error {
	return fmt.Errorf("eject not supported on this platform")
}
