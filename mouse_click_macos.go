//go:build darwin
// +build darwin

// This file contains functions that use Core Graphics (via cgo) to simulate mouse events.
package main

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>
#include <stdlib.h>

// postMouseClick simulates moving the mouse to (x,y) and then performing a left-click.
void postMouseClick(int x, int y) {
    // Move the mouse pointer to the coordinate.
    CGEventRef move = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, CGPointMake(x, y), kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, move);
    CFRelease(move);

    // Simulate mouse button press.
    CGEventRef clickDown = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseDown, CGPointMake(x, y), kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, clickDown);
    CFRelease(clickDown);

    // Simulate mouse button release.
    CGEventRef clickUp = CGEventCreateMouseEvent(NULL, kCGEventLeftMouseUp, CGPointMake(x, y), kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, clickUp);
    CFRelease(clickUp);
}

// moveMouse moves the mouse pointer to the specified coordinates without clicking.
void moveMouse(int x, int y) {
    CGEventRef move = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, CGPointMake(x, y), kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, move);
    CFRelease(move);
}
*/
import "C"
import "image"

// clickAtAbsolutePointCG moves the mouse to the absolute coordinate and performs a click using Core Graphics.
func clickAtAbsolutePointCG(pt image.Point) {
	C.postMouseClick(C.int(pt.X), C.int(pt.Y))
}

// moveMouseToCoordinateCG moves the mouse pointer (without clicking) to the specified absolute coordinate.
func moveMouseToCoordinateCG(pt image.Point) {
	C.moveMouse(C.int(pt.X), C.int(pt.Y))
}