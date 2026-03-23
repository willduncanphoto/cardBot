package analyze

import (
	"testing"
)

func TestIsPhoto(t *testing.T) {
	if !IsPhoto("NEF") {
		t.Error("Expected NEF to be a photo")
	}
	if IsPhoto("MOV") {
		t.Error("Expected MOV not to be a photo")
	}
}

func TestIsVideo(t *testing.T) {
	if !IsVideo("MOV") {
		t.Error("Expected MOV to be a video")
	}
	if IsVideo("NEF") {
		t.Error("Expected NEF not to be a video")
	}
}
