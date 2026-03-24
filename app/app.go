// Package app contains the core CardBot application logic.
package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/illwill/cardbot/analyze"
	"github.com/illwill/cardbot/cblog"
	"github.com/illwill/cardbot/config"
	"github.com/illwill/cardbot/detect"
)

// UX delays — gives the user time to read each startup line before the next appears.
const (
	removalDelay = 2 * time.Second // Pause after card removal so message is visible
)

// App is the main CardBot application.
type App struct {
	ctx         context.Context
	detector    cardDetector
	currentCard *detect.Card
	lastResult  *analyze.Result // analysis result for currentCard
	cardQueue   []*detect.Card
	mu          sync.Mutex
	printMu     sync.Mutex // serialises concurrent stdout writes during copy
	cfg         *config.Config
	logger      *cblog.Logger
	inputChan   chan string // buffered input from stdin
	dryRun      bool
	copiedModes map[string]bool    // modes completed this session
	cardInvalid bool               // true when current card has no DCIM directory
	scanCancel  context.CancelFunc // cancels the current displayCard goroutine
	spinner     *spinner.Spinner   // scanner spinner
	version     string             // app version for display and dotfile
	phase       appPhase           // explicit runtime phase
	targetPath  string             // optional: skip scanning and target this path directly
	lastTS      string             // last printed timestamp for indentation

	newDetector  detectorFactory
	newAnalyzer  analyzerFactory
	runCopy      copyRunner
	writeDotfile dotfileWriter
}

// Config holds the configuration for creating a new App.
type Config struct {
	Cfg        *config.Config
	Logger     *cblog.Logger
	DryRun     bool
	Version    string
	TargetPath string // optional: skip scanning and target this path directly

	// Optional dependency overrides for tests.
	newDetector  detectorFactory
	newAnalyzer  analyzerFactory
	runCopy      copyRunner
	writeDotfile dotfileWriter
}

// New creates a new App instance.
func New(c Config) *App {
	inputChan := make(chan string, 10)

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
		ctx:          context.Background(), // default; overridden by Run()
		cardQueue:    make([]*detect.Card, 0),
		cfg:          c.Cfg,
		logger:       c.Logger,
		inputChan:    inputChan,
		dryRun:       c.DryRun,
		copiedModes:  make(map[string]bool),
		version:      c.Version,
		phase:        phaseScanning,
		targetPath:   normalizeCardPath(c.TargetPath),
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

// tsIndent is whitespace matching the width of a "[2006-01-02T15:04:05]" timestamp.
const tsIndent = "                     "

// dimTS returns a dimmed timestamp string using ANSI escape codes.
func dimTS(ts string) string {
	return "\033[2m[" + ts + "]\033[0m"
}

// TsPrefix returns a bracketed timestamp prefix for the current second.
// If the current second matches the last printed timestamp, it returns
// whitespace of the same width so subsequent lines stay aligned.
func (a *App) TsPrefix() string {
	now := Ts()
	if now == a.lastTS {
		return tsIndent
	}
	a.lastTS = now
	return dimTS(now)
}

// SetLastTS records a timestamp so that TsPrefix can deduplicate it.
// Used by main.go to sync the bootup timestamp with the app.
func (a *App) SetLastTS(t string) {
	a.lastTS = t
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

// Run starts the main event loop. It blocks until the context is cancelled.
func (a *App) Run(ctx context.Context) error {
	a.ctx = ctx
	a.detector = a.newDetector()
	if err := a.detector.Start(); err != nil {
		return err
	}
	defer a.detector.Stop()

	go readInput(ctx, a.inputChan)

	// If a target path was specified, synthesize a card event immediately.
	if a.targetPath != "" {
		a.launchTargetPath(a.targetPath)
	}

	for {
		select {
		case card := <-a.detector.Events():
			a.handleCardEvent(card)

		case path := <-a.detector.Removals():
			a.handleRemoval(path)

		case input := <-a.inputChan:
			a.handleInput(input)

		case <-ctx.Done():
			a.setPhase(phaseShuttingDown)
			a.stopScanning()
			fmt.Println("\nShutting down...")
			a.logf("Shutting down")
			return nil
		}
	}
}

// StartScanning starts the scanning spinner.
func (a *App) StartScanning() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.startScanningLocked()
}

func (a *App) startScanningLocked() {
	if a.spinner != nil {
		a.spinner.Stop()
	}
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = fmt.Sprintf("[%s] Scanning ", Ts())
	s.Start()
	a.spinner = s
	a.setPhaseLocked(phaseScanning)
}

// stopScanning stops and clears the scanning spinner.
func (a *App) stopScanning() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopScanningLocked()
}

func (a *App) stopScanningLocked() {
	if a.spinner != nil {
		a.spinner.Stop()
		a.spinner = nil
	}
}

func readInput(ctx context.Context, ch chan<- string) {
	reader := newStdinReader()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		select {
		case ch <- strings.TrimSpace(line):
		case <-ctx.Done():
			return
		}
	}
}
