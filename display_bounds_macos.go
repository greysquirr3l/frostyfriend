//go:build darwin
// +build darwin

// Package main contains macOS-specific code.
// This file computes the union of all active display bounds using Core Graphics.
package main

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>
*/
import "C"
import "fmt"

// getUnionDisplayBounds computes the union (the smallest rectangle that encloses all displays)
// of all active displays on macOS. It returns a Rect containing the global origin and size.
func getUnionDisplayBounds() (Rect, error) {
	var maxDisplays C.uint32_t = 16
	displayIDs := make([]C.CGDirectDisplayID, maxDisplays)
	var displayCount C.uint32_t

	// Retrieve the list of active displays.
	if C.CGGetActiveDisplayList(maxDisplays, &displayIDs[0], &displayCount) != C.kCGErrorSuccess {
		return Rect{}, fmt.Errorf("failed to get active display list")
	}

	if displayCount == 0 {
		return Rect{}, fmt.Errorf("no active displays found")
	}

	// Initialize union using the bounds of the first display.
	bounds := C.CGDisplayBounds(displayIDs[0])
	minX := int(bounds.origin.x)
	minY := int(bounds.origin.y)
	maxX := minX + int(bounds.size.width)
	maxY := minY + int(bounds.size.height)

	// Loop through the remaining displays and update the union.
	for i := 1; i < int(displayCount); i++ {
		b := C.CGDisplayBounds(displayIDs[i])
		x := int(b.origin.x)
		y := int(b.origin.y)
		w := int(b.size.width)
		h := int(b.size.height)
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x+w > maxX {
			maxX = x + w
		}
		if y+h > maxY {
			maxY = y + h
		}
	}

	return Rect{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}, nil
}