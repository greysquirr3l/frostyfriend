package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gocv.io/x/gocv"
)

type ClickTarget struct {
	Name string
	X    float64 // Percentage of window width
	Y    float64 // Percentage of window height
}

// THESE ARE GOLDEN VALUES.  NEVER CHANGE THEM
var clickTarget = ClickTarget{
	Name: "Handshake",
	X:    1.40, // Adjust this value to move the click point horizontally
	Y:    1.78, // Adjust this value to move the click point vertically
}

//

// IsAppRunning checks if the specified app is running
func IsAppRunning(appName string) bool {
	log.Printf("IsAppRunning")
	cmd := exec.Command("osascript", "-e", `
        tell application "System Events"
            set appList to (name of every process)
            return appList
        end tell
    `)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("Error detecting application: %v", err)
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(appName))
}

// LocateWindow finds the position and size of the application window using AppleScript
func LocateWindow(appName string) (int, int, int, int, error) {
	log.Printf("LocateWindow")
	script := `
        tell application "System Events"
            set appProcess to first process whose name is "` + appName + `"
            set appWindow to first window of appProcess
            set {x, y} to position of appWindow
            set {width, height} to size of appWindow
            return x & "," & y & "," & width & "," & height
        end tell
    `
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		log.Printf("Error locating window: %v", err)
		return 0, 0, 0, 0, fmt.Errorf("Could not locate window: %v", err)
	}

	// Update: Change the delimiter to comma
	dimensions := strings.TrimSpace(string(out))
	// Remove any extra spaces or commas
	dimensionSlice := strings.FieldsFunc(dimensions, func(r rune) bool {
		return r == ',' || r == ' ' // Split on commas and spaces
	})

	if len(dimensionSlice) != 4 {
		log.Printf("Raw output: %s", dimensions)
		return 0, 0, 0, 0, fmt.Errorf("Unexpected number of dimensions: %v", dimensionSlice)
	}

	var x, y, width, height int
	var parseErr error

	// Attempt to parse coordinates
	x, parseErr = strconv.Atoi(strings.TrimSpace(dimensionSlice[0]))
	if parseErr != nil {
		return 0, 0, 0, 0, fmt.Errorf("Failed to parse x coordinate: %v", parseErr)
	}

	y, parseErr = strconv.Atoi(strings.TrimSpace(dimensionSlice[1]))
	if parseErr != nil {
		return 0, 0, 0, 0, fmt.Errorf("Failed to parse y coordinate: %v", parseErr)
	}

	width, parseErr = strconv.Atoi(strings.TrimSpace(dimensionSlice[2]))
	if parseErr != nil {
		return 0, 0, 0, 0, fmt.Errorf("Failed to parse width: %v", parseErr)
	}

	height, parseErr = strconv.Atoi(strings.TrimSpace(dimensionSlice[3]))
	if parseErr != nil {
		return 0, 0, 0, 0, fmt.Errorf("Failed to parse height: %v", parseErr)
	}

	log.Printf("Window found: (%d, %d) with size (%d, %d)\n", x, y, width, height)

	return x, y, width, height, nil
}

// FocusApp brings the specified application to the foreground
func FocusApp(appName string) {
	log.Printf("FocusApp")
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to activate`, appName))
	if err := cmd.Run(); err != nil {
		log.Printf("Error focusing application %s: %v", appName, err)
	} else {
		log.Printf("Application %s brought to foreground", appName)
	}
}

// CaptureScreen takes a screenshot of the specified region
func CaptureScreen(windowX, windowY, width, height int) gocv.Mat {
	log.Printf("CaptureScreen")
	outputPath := "/tmp/screenshot.png"
	cmd := exec.Command("screencapture", "-R", fmt.Sprintf("%d,%d,%d,%d", windowX, windowY, width, height), outputPath)
	if err := cmd.Run(); err != nil {
		log.Printf("Error capturing screen: %v", err)
		return gocv.NewMat()
	}

	img := gocv.IMRead(outputPath, gocv.IMReadColor)
	if img.Empty() {
		log.Println("Failed to load screenshot")
		return gocv.NewMat()
	}

	log.Printf("Screenshot captured successfully. Size: %dx%d", img.Cols(), img.Rows())
	return img
}

// Update the ClickInWindow function signature
func ClickInWindow(windowX, windowY, width, height, screenshotWidth, screenshotHeight int, target ClickTarget) {
	log.Printf("ClickInWindow")
	// Calculate the scaling factors
	scaleX := float64(screenshotWidth) / float64(width)
	scaleY := float64(screenshotHeight) / float64(height)

	// Calculate click position using the scaling factors
	clickX := windowX + int(float64(width)*target.X/scaleX)
	clickY := windowY + int(float64(height)*target.Y/scaleY)

	log.Printf("Clicking at position: (%d, %d)", clickX, clickY)

	cmd := exec.Command("cliclick", fmt.Sprintf("c:%d,%d", clickX, clickY))
	if err := cmd.Run(); err != nil {
		log.Printf("Error performing click: %v", err)
	} else {
		log.Printf("Click performed at position: (%d, %d)", clickX, clickY)
	}
}

func SearchAndClick(template gocv.Mat, screen gocv.Mat, windowX, windowY, windowWidth, windowHeight int) bool {
	log.Printf("SearchAndClick")
	result := gocv.NewMat()
	defer result.Close()

	log.Printf("Screen size: (%d, %d), Template size: (%d, %d)", screen.Cols(), screen.Rows(), template.Cols(), template.Rows())

	gocv.MatchTemplate(screen, template, &result, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)

	if maxVal >= 0.75 {
		log.Printf("Template found with match value: %.2f at position: (%d, %d)", maxVal, maxLoc.X, maxLoc.Y)

		// Calculate the center of the matched area
		centerX := maxLoc.X + (template.Cols() / 2)
		centerY := maxLoc.Y + (template.Rows() / 2)

		// Scale the coordinates
		scaleX := float64(windowWidth) / float64(screen.Cols())
		scaleY := float64(windowHeight) / float64(screen.Rows())

		// Adjust the coordinates relative to the window
		adjustedX := windowX + int(float64(centerX)*scaleX)
		adjustedY := windowY + int(float64(centerY)*scaleY)

		log.Printf("Clicking at adjusted position: (%d, %d)", adjustedX, adjustedY)

		// Draw a green rectangle around the matched area (for debugging)
		// rect := image.Rect(maxLoc.X, maxLoc.Y, maxLoc.X+template.Cols(), maxLoc.Y+template.Rows())
		// gocv.Rectangle(&screen, rect, color.RGBA{0, 255, 0, 255}, 2)

		// Draw a small green circle where the click is supposed to happen
		// gocv.Circle(&screen, image.Pt(centerX, centerY), 5, color.RGBA{0, 255, 0, 255}, -1)

		// outputFilePath := "./matched_result_with_dot.png"
		// if ok := gocv.IMWrite(outputFilePath, screen); !ok {
		//	log.Printf("Error saving the result image to %s", outputFilePath)
		// }
		// log.Printf("Result with dot saved to %s", outputFilePath)

		// Perform the click
		cmd := exec.Command("cliclick", fmt.Sprintf("c:%d,%d", adjustedX, adjustedY))
		if err := cmd.Run(); err != nil {
			log.Printf("Error performing click: %v", err)
			return false
		}
		log.Printf("Click performed at position: (%d, %d)", adjustedX, adjustedY)
		return true
	}

	log.Printf("Template not found. Max match value: %.2f", maxVal)
	outputFilePath := "./matched_result_nomatch.png"
	if ok := gocv.IMWrite(outputFilePath, screen); !ok {
		log.Printf("Error saving the result image to %s", outputFilePath)
	}
	log.Printf("Result without match saved to %s", outputFilePath)

	return false
}

func main() {
	log.SetOutput(os.Stdout)
	log.Println("Starting Whiteout Survival helper")

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChannel
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	appName := "WhiteoutSurvival"

	// Load the template image from the images folder
	templatePath := filepath.Join("images", "handshake_icon.png")
	template := gocv.IMRead(templatePath, gocv.IMReadColor)
	if template.Empty() {
		log.Fatalf("Error reading template image file: %s", templatePath)
	}
	defer template.Close()

	log.Printf("Template image loaded successfully from: %s", templatePath)

	for {
		log.Println("Starting iteration")
		if IsAppRunning(appName) {
			FocusApp(appName)
			time.Sleep(500 * time.Millisecond)

			x, y, width, height, err := LocateWindow(appName)
			if err != nil {
				log.Printf("Error locating window: %v", err)
				continue
			}

			log.Printf("Monitoring window at position: (%d, %d), size: (%d, %d)", x, y, width, height)

			screen := CaptureScreen(x, y, width, height)
			if screen.Empty() {
				log.Println("Failed to capture screen")
				continue
			}

			// Use SearchAndClick with all necessary parameters
			matched := SearchAndClick(template, screen, x, y, width, height)

			if matched {
				log.Println("Handshake icon found and clicked")
			} else {
				log.Println("Handshake icon not found in this iteration")
			}

			screen.Close()
		} else {
			log.Println("Application not running")
		}

		log.Println("Iteration complete, waiting for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}
