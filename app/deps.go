package app

import (
	"context"

	"github.com/illwill/cardbot/analyze"
	"github.com/illwill/cardbot/cardcopy"
	"github.com/illwill/cardbot/detect"
	"github.com/illwill/cardbot/dotfile"
)

// cardDetector is the app-facing detector contract.
// Defined at the consumer side to keep integration points testable.
type cardDetector interface {
	Start() error
	Stop()
	Events() <-chan *detect.Card
	Removals() <-chan string
	Eject(path string) error
	Remove(path string)
}

// cardAnalyzer is the app-facing analyzer contract.
type cardAnalyzer interface {
	SetWorkers(n int)
	OnProgress(fn analyze.ProgressFunc)
	Analyze(ctx context.Context) (*analyze.Result, error)
}

type detectorFactory func() cardDetector
type analyzerFactory func(cardPath string) cardAnalyzer
type copyRunner func(ctx context.Context, opts cardcopy.Options, onProgress cardcopy.ProgressFunc) (*cardcopy.Result, error)
type dotfileWriter func(opts dotfile.WriteOptions) error

var (
	defaultDetectorFactory detectorFactory = func() cardDetector { return detect.NewDetector() }
	defaultAnalyzerFactory analyzerFactory = func(cardPath string) cardAnalyzer { return analyze.New(cardPath) }
	defaultCopyRunner      copyRunner      = cardcopy.Run
	defaultDotfileWriter   dotfileWriter   = dotfile.Write
)
