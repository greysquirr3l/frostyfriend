//go:build darwin
// +build darwin

// This file contains macOS-specific helper functions using AppleScript to interact with applications.
package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gocv.io/x/gocv"
)

// focusApp uses AppleScript to ensure that the application (appName) is not minimized and is brought to the front.
func focusApp() {
	// This AppleScript activates the app and ensures its window is not minimized.
	script := fmt.Sprintf(`
		tell application "%s" to activate
		tell application "System Events"
			tell process "%s"
				try
					if miniaturized of window 1 is true then
						set miniaturized of window 1 to false
					end if
				end try
				set frontmost to true
			end tell
		end tell
	`, appName, appName)
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		logChan <- fmt.Sprintf("Error focusing application %s: %v", appName, err)
	}
}


//* getAppWindowInfo retrieves the global position and size of the application's window using AppleScript.
//It performs the following steps:
// 1. Checks if the process (with name appName) exists.
// 2. Retrieves the count of windows for the process.
//     - If the count is 0 (or "NO_WINDOW"), it activates the application to force it to show a window.
// 3. Polls every 500 milliseconds (up to a timeout of 3 seconds) for the window to become available.
// 4. Once a window is available, it retrieves its position and size (as text), converts these to integers,
//    and returns a Rect containing the window's global coordinates.
//     
//If the application is not running or no window becomes available within 3 seconds, the function returns an error.
//This polling mechanism is useful for apps that temporarily close or hide their window when idle.
//*/
func getAppWindowInfo() (Rect, error) {
	// tryGetAppWindowInfo encapsulates the AppleScript call to get window info.
	tryGetAppWindowInfo := func() (Rect, error) {
		script := fmt.Sprintf(`
			tell application "System Events"
				if exists (processes whose name is "%s") then
					tell process "%s"
						set winCount to count of windows
						if winCount = 0 then
							return "NO_WINDOW"
						else
							set theWindow to first window
							set pos to position of theWindow
							set sz to size of theWindow
							return ((item 1 of pos) as text) & "," & ((item 2 of pos) as text) & "," & ((item 1 of sz) as text) & "," & ((item 2 of sz) as text)
						end if
					end tell
				else
					return "NOTFOUND"
				end if
			end tell
		`, appName, appName)

		cmd := exec.Command("osascript", "-e", script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return Rect{}, fmt.Errorf("AppleScript error: %v, output: %s", err, string(out))
		}
		result := strings.TrimSpace(string(out))
		if result == "NOTFOUND" {
			return Rect{}, errors.New("application not running")
		}
		if result == "NO_WINDOW" {
			return Rect{}, errors.New("window not available")
		}
		parts := strings.Split(result, ",")
		if len(parts) != 4 {
			return Rect{}, fmt.Errorf("unexpected window info: %s", result)
		}
		x, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		y, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		w, err3 := strconv.Atoi(strings.TrimSpace(parts[2]))
		h, err4 := strconv.Atoi(strings.TrimSpace(parts[3]))
		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			return Rect{}, fmt.Errorf("error parsing window info: %s", result)
		}
		return Rect{X: x, Y: y, Width: w, Height: h}, nil
	}

	// First, check if any window is available.
	checkScript := fmt.Sprintf(`
		tell application "System Events"
			if exists (processes whose name is "%s") then
				tell process "%s" to count windows
			else
				return "NOTFOUND"
			end if
		end tell
	`, appName, appName)
	countCmd := exec.Command("osascript", "-e", checkScript)
	countOut, _ := countCmd.CombinedOutput()
	countStr := strings.TrimSpace(string(countOut))
	if countStr == "0" || countStr == "NO_WINDOW" {
		logChan <- fmt.Sprintf("No window available; activating %s...", appName)
		activateCmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to activate`, appName))
		activateCmd.Run() // Ignore errors here.
	}

	// Poll every 500ms, but timeout after 3 seconds.
	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return Rect{}, errors.New("timeout waiting for window to appear")
		case <-ticker.C:
			if rect, err := tryGetAppWindowInfo(); err == nil {
				logChan <- fmt.Sprintf("Found %s window: %+v", appName, rect)
				return rect, nil
			} else {
				logChan <- fmt.Sprintf("Window not ready: %v", err)
			}
		}
	}
}

// captureScreen uses the macOS 'screencapture' command to take a screenshot of the given window region.
// It returns the image as a gocv.Mat.
func captureScreen(rect Rect) (gocv.Mat, error) {
	outputPath := "/tmp/screenshot.png"
	if rect.Width <= 0 || rect.Height <= 0 {
		logChan <- fmt.Sprintf("Invalid dimensions: width=%d, height=%d", rect.Width, rect.Height)
		return gocv.NewMat(), fmt.Errorf("invalid dimensions")
	}
	region := fmt.Sprintf("%d,%d,%d,%d", rect.X, rect.Y, rect.Width, rect.Height)
	cmd := exec.Command("screencapture", "-x", "-R", region, outputPath)
	if err := cmd.Run(); err != nil {
		logChan <- fmt.Sprintf("Error capturing screen: %v", err)
		return gocv.NewMat(), err
	}
	img := gocv.IMRead(outputPath, gocv.IMReadColor)
	if img.Empty() {
		logChan <- "Failed to load screenshot"
		return gocv.NewMat(), fmt.Errorf("empty screenshot")
	}
	return img, nil
}
