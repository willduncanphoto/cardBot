// Package app contains the core CardBot application logic.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/briandowns/spinner"
	"github.com/illwill/cardbot/internal/analyze"
	"github.com/illwill/cardbot/internal/config"
	"github.com/illwill/cardbot/internal/detect"
	cblog "github.com/illwill/cardbot/internal/log"
)

// UX delays — gives the user time to read each startup line before the next appears.
const (
	removalDelay = 2 * time.Second // Pause after card removal so message is visible
)

// App is the main CardBot application.
type App struct {
	detector    cardDetector
	currentCard *detect.Card
	lastResult  *analyze.Result // analysis result for currentCard
	cardQueue   []*detect.Card
	mu          sync.Mutex
	printMu     sync.Mutex // serialises concurrent stdout writes during copy
	cfg         *config.Config
	logger      *cblog.Logger
	inputChan   chan string    // buffered input from stdin
	sigChan     chan os.Signal // SIGINT/SIGTERM
	inputDone   chan struct{}  // closed on shutdown to stop readInput
	dryRun      bool
	copiedModes map[string]bool    // modes completed this session
	cardInvalid bool               // true when current card has no DCIM directory
	scanCancel  context.CancelFunc // cancels the current displayCard goroutine
	spinner     *spinner.Spinner   // scanner spinner
	version     string             // app version for display and dotfile
	phase       appPhase           // explicit runtime phase

	newDetector  detectorFactory
	newAnalyzer  analyzerFactory
	runCopy      copyRunner
	writeDotfile dotfileWriter
}

// Config holds the configuration for creating a new App.
type Config struct {
	Cfg     *config.Config
	Logger  *cblog.Logger
	DryRun  bool
	Version string

	// Optional dependency overrides for tests.
	newDetector  detectorFactory
	newAnalyzer  analyzerFactory
	runCopy      copyRunner
	writeDotfile dotfileWriter
}

// New creates a new App instance.
func New(c Config) *App {
	inputChan := make(chan string, 10)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	inputDone := make(chan struct{})

	newDetector := c.newDetector
	if newDetector == nil {
		newDetector = defaultDetectorFactory
	}
	newAnalyzer := c.newAnalyzer
	if newAnalyzer == nil {
		newAnalyzer = defaultAnalyzerFactory
	}
	runCopy := c.runCopy
	if runCopy == nil {
		runCopy = defaultCopyRunner
	}
	writeDotfile := c.writeDotfile
	if writeDotfile == nil {
		writeDotfile = defaultDotfileWriter
	}

	return &App{
		cardQueue:    make([]*detect.Card, 0),
		cfg:          c.Cfg,
		logger:       c.Logger,
		inputChan:    inputChan,
		sigChan:      sigChan,
		inputDone:    inputDone,
		dryRun:       c.DryRun,
		copiedModes:  make(map[string]bool),
		version:      c.Version,
		phase:        phaseScanning,
		newDetector:  newDetector,
		newAnalyzer:  newAnalyzer,
		runCopy:      runCopy,
		writeDotfile: writeDotfile,
	}
}

// Ts returns the current timestamp formatted for log output.
func Ts() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

// ts is an internal alias for Ts.
func ts() string {
	return Ts()
}

// logf writes to the log file if logging is enabled, and is a no-op otherwise.
func (a *App) logf(format string, args ...any) {
	if a.logger != nil {
		a.logger.Printf(format, args...)
	}
}

// Printf prints to stdout and mirrors to the log file (without adding a second timestamp).
func (a *App) Printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(msg)
	if a.logger != nil {
		// Caller already includes [timestamp] in the message, so write raw.
		a.logger.Raw(strings.TrimRight(msg, "\n"))
	}
}

// drainInput discards any buffered input keystrokes.
// Called after blocking operations (copy, speed test) to prevent
// queued commands from firing on the next prompt.
func (a *App) drainInput() {
	for {
		select {
		case <-a.inputChan:
		default:
			return
		}
	}
}

// Run starts the main event loop. It blocks until shutdown.
func (a *App) Run() error {
	a.detector = a.newDetector()
	if err := a.detector.Start(); err != nil {
		return err
	}
	defer a.detector.Stop()

	go readInput(a.inputChan, a.inputDone)

	for {
		select {
		case card := <-a.detector.Events():
			a.handleCardEvent(card)

		case path := <-a.detector.Removals():
			a.handleRemoval(path)

		case input := <-a.inputChan:
			a.handleInput(input)

		case <-a.sigChan:
			a.setPhase(phaseShuttingDown)
			a.stopScanning()
			fmt.Println("\nShutting down...")
			a.logf("Shutting down")
			close(a.inputDone)
			return nil
		}
	}
}

// StartScanning starts the scanning spinner.
func (a *App) StartScanning() {
	if a.spinner != nil {
		a.spinner.Stop()
	}
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = fmt.Sprintf("[%s] Scanning ", ts())
	s.Start()
	a.spinner = s
	a.setPhase(phaseScanning)
}

// stopScanning stops and clears the scanning spinner.
func (a *App) stopScanning() {
	if a.spinner != nil {
		a.spinner.Stop()
		a.spinner = nil
	}
}

func readInput(ch chan<- string, done <-chan struct{}) {
	reader := newStdinReader()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		select {
		case ch <- strings.TrimSpace(line):
		case <-done:
			return
		}
	}
}
