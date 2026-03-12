package ui

// BrandColor returns an ANSI color code for the given camera brand.
// Falls back to white for unknown brands.
func BrandColor(brand string) string {
	switch brand {
	case "Nikon":
		return "\033[33m" // Yellow
	case "Canon":
		return "\033[31m" // Red
	case "Sony":
		return "\033[37m" // White
	case "Fujifilm":
		return "\033[32m" // Green
	case "Panasonic":
		return "\033[34m" // Blue
	case "Olympus":
		return "\033[36m" // Cyan
	default:
		return "\033[37m" // White
	}
}

// Reset is the ANSI reset code.
const Reset = "\033[0m"
