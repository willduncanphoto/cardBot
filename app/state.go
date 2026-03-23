package app

// appPhase is the high-level runtime state of the app.
// It centralizes command/readiness flow while coexisting with card-specific
// fields (currentCard, copiedModes, cardInvalid).
type appPhase int

const (
	phaseScanning appPhase = iota
	phaseAnalyzing
	phaseReady
	phaseCopying
	phaseShuttingDown
)

func (p appPhase) String() string {
	switch p {
	case phaseScanning:
		return "scanning"
	case phaseAnalyzing:
		return "analyzing"
	case phaseReady:
		return "ready"
	case phaseCopying:
		return "copying"
	case phaseShuttingDown:
		return "shutting_down"
	default:
		return "unknown"
	}
}

func (a *App) setPhase(p appPhase) {
	a.mu.Lock()
	a.setPhaseLocked(p)
	a.mu.Unlock()
}

func (a *App) setPhaseLocked(p appPhase) {
	if !canTransitionPhase(a.phase, p) {
		return
	}
	a.phase = p
}

func (a *App) currentPhase() appPhase {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.phase
}

// canTransitionPhase enforces a small, explicit phase transition table.
func canTransitionPhase(from, to appPhase) bool {
	if from == to {
		return true
	}
	if to == phaseShuttingDown {
		return true
	}
	if from == phaseShuttingDown {
		return false
	}

	switch from {
	case phaseScanning:
		return to == phaseAnalyzing
	case phaseAnalyzing:
		return to == phaseReady || to == phaseScanning
	case phaseReady:
		return to == phaseCopying || to == phaseScanning || to == phaseAnalyzing
	case phaseCopying:
		return to == phaseReady || to == phaseScanning || to == phaseAnalyzing
	default:
		return false
	}
}

func phaseAfterFinish(queueLen int) appPhase {
	if queueLen > 0 {
		return phaseAnalyzing
	}
	return phaseScanning
}

// finishCopyPhase restores the phase after a copy command returns.
// It only flips to ready if the same card is still active.
func (a *App) finishCopyPhase(cardPath string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.phase != phaseCopying {
		return
	}
	if a.currentCard == nil || a.currentCard.Path != cardPath {
		return
	}
	a.phase = phaseReady
}
