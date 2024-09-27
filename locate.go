package locate

import (
	"fmt"
	"log"
	"os/exec"
)

// IsAppRunning checks if the Whiteout Survival game is running using AppleScript
func IsAppRunning(appName string) bool {
	script := `
        tell application "System Events"
            count (every process whose name is "` + appName + `")
        end tell
    `
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("Error detecting application: %v", err)
		return false
	}

	// If the output is "0", the app is not running
	if string(out) == "0\n" {
		log.Printf("%s is not running", appName)
		return false
	}
	log.Printf("%s is running", appName)
	return true
}

// LocateWindow finds the position of the application window using AppleScript
func LocateWindow(appName string) (int, int, int, int, error) {
	script := `
        tell application "System Events"
            get position of front window of application process "` + appName + `"
        end tell
    `
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("Could not locate window: %v", err)
	}

	var x, y, width, height int
	fmt.Sscanf(string(out), "{%d, %d}", &x, &y)

	// Simulate window size for now
	width, height = 800, 600

	// Log only when the window is successfully found
	log.Printf("Window found: (%d, %d) with size (%d, %d)\n", x, y, width, height)

	return x, y, width, height, nil
}
