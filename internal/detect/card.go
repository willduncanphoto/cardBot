package detect

// Card represents a detected memory card.
type Card struct {
	Path       string // Mount point (e.g., /Volumes/NIKON Z9)
	Name       string // Volume name
	TotalBytes int64  // Total capacity (filesystem view)
	UsedBytes  int64  // Used space
	Brand      string // Guessed brand from DCIM folder naming
	Hardware   *HardwareInfo // Hardware-level info (may be nil if unavailable)
}
