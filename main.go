// main.go - revision 9

package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"image"
	"image/color"

	"gocv.io/x/gocv"
)

type ClickTarget struct {
	Name string
	X    float64 // Percentage of window width
	Y    float64 // Percentage of window height
}

var clickTarget = ClickTarget{
	Name: "Handshake",
	X:    1.40,
	Y:    1.78,
}

var debugMode bool
var logChan = make(chan string, 100)
var totalClicks int

func logRoutine() {
	for logMsg := range logChan {
		log.Println(logMsg)
	}
}

func IsAppRunningAndLocateWindow(appName string) (bool, int, int, int, int, error) {
	script := `
        tell application "System Events"
            set appList to (name of every process)
            if "` + appName + `" is in appList then
                set appProcess to first process whose name is "` + appName + `"
                set appWindow to first window of appProcess
                set {x, y} to position of appWindow
                set {width, height} to size of appWindow
                return "true," & x & "," & y & "," & width & "," & height
            else
                return "false"
            end if
        end tell
    `
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return false, 0, 0, 0, 0, fmt.Errorf("could not locate window: %v", err)
	}

	output := strings.TrimSpace(string(out))
	if strings.HasPrefix(output, "false") {
		return false, 0, 0, 0, 0, nil
	}

	dimensionSlice := strings.FieldsFunc(output[5:], func(r rune) bool {
		return r == ',' || r == ' '
	})

	if len(dimensionSlice) != 4 {
		return false, 0, 0, 0, 0, fmt.Errorf("unexpected number of dimensions: %v", dimensionSlice)
	}

	x, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[0]))
	y, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[1]))
	width, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[2]))
	height, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[3]))

	return true, x, y, width, height, nil
}

func FocusApp(appName string) {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to activate`, appName))
	if err := cmd.Run(); err != nil {
		logChan <- fmt.Sprintf("Error focusing application %s: %v", appName, err)
	}
}

func CaptureScreen(windowX, windowY, width, height int, screenChan chan gocv.Mat, wg *sync.WaitGroup) {
	defer wg.Done()
	outputPath := "/tmp/screenshot.png"
	cmd := exec.Command("screencapture", "-R", fmt.Sprintf("%d,%d,%d,%d", windowX, windowY, width, height), outputPath)

	if err := cmd.Run(); err != nil {
		logChan <- fmt.Sprintf("Error capturing screen: %v", err)
		screenChan <- gocv.NewMat()
		return
	}

	img := gocv.IMRead(outputPath, gocv.IMReadColor)
	if img.Empty() {
		logChan <- "Failed to load screenshot"
		screenChan <- gocv.NewMat()
		return
	}

	screenChan <- img
}

func SearchAndClick(template gocv.Mat, screen gocv.Mat, windowX, windowY, windowWidth, windowHeight int, targetName string) bool {
	result := gocv.NewMat()
	defer result.Close()

	gocv.MatchTemplate(screen, template, &result, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)

	if maxVal >= 0.75 {
		centerX := maxLoc.X + (template.Cols() / 2)
		centerY := maxLoc.Y + (template.Rows() / 2)

		scaleX := float64(windowWidth) / float64(screen.Cols())
		scaleY := float64(windowHeight) / float64(screen.Rows())

		adjustedX := windowX + int(float64(centerX)*scaleX)
		adjustedY := windowY + int(float64(centerY)*scaleY)

		if debugMode {
			// Draw blue rectangle around the matched region for big_x.png, green for others
			rectColor := color.RGBA{0, 255, 0, 0} // Green for handshake_icon.png
			if targetName == "big_x" {
				rectColor = color.RGBA{0, 0, 255, 0} // Blue for big_x.png
			}
			rect := image.Rect(maxLoc.X, maxLoc.Y, maxLoc.X+template.Cols(), maxLoc.Y+template.Rows())
			gocv.Rectangle(&screen, rect, rectColor, 2)

			// Draw red circle where the click will take place
			gocv.Circle(&screen, image.Pt(centerX, centerY), 5, color.RGBA{255, 0, 0, 0}, -1)

			// Save the debug screenshot
			debugPath := fmt.Sprintf("%s-debug.png", targetName)
			if ok := gocv.IMWrite(debugPath, screen); !ok {
				logChan <- fmt.Sprintf("Error saving debug screenshot: %s", debugPath)
			} else {
				logChan <- fmt.Sprintf("Debug screenshot saved: %s", debugPath)
			}
		}

		cmd := exec.Command("cliclick", fmt.Sprintf("c:%d,%d", adjustedX, adjustedY))
		if err := cmd.Run(); err != nil {
			logChan <- fmt.Sprintf("Error performing click: %v", err)
			return false
		}
		totalClicks++
		logChan <- fmt.Sprintf("Click performed on %s at position: (%d, %d) (Total clicks: %d)", targetName, adjustedX, adjustedY, totalClicks)
		return true
	}
	return false
}

func main() {
	// Define flags
	iterationDelay := flag.Int("delay", 10, "Delay between iterations in seconds")
	randomDelay := flag.Bool("random", false, "Use random delay between 0 and specified delay")
	iterationCount := flag.Int("iterations", 0, "Number of iterations to run (0 for infinite)")
	helpFlag := flag.Bool("help", false, "Display help information")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode to save annotated screenshots")

	// Custom usage function to display help
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Whiteout Survival Helper - Automates interactions with the game.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	// Parse flags
	flag.Parse()

	// Check if help flag is set
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	log.SetOutput(os.Stdout)
	go logRoutine()
	logChan <- "Starting Whiteout Survival helper"

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChannel
		logChan <- "Shutting down..."
		close(logChan)
		os.Exit(0)
	}()

	handshakePath := filepath.Join("images", "handshake_icon.png")
	handshakeTemplate := gocv.IMRead(handshakePath, gocv.IMReadColor)
	if handshakeTemplate.Empty() {
		log.Fatalf("Error reading template image file: %s", handshakePath)
	}
	defer handshakeTemplate.Close()
	logChan <- fmt.Sprintf("Template image loaded successfully from: %s", handshakePath)

	bigXPath := filepath.Join("images", "big_x.png")
	bigXTemplate := gocv.IMRead(bigXPath, gocv.IMReadColor)
	if bigXTemplate.Empty() {
		log.Fatalf("Error reading template image file: %s", bigXPath)
	}
	defer bigXTemplate.Close()
	logChan <- fmt.Sprintf("Template image loaded successfully from: %s", bigXPath)

	// Initialize random number generator if random delay is enabled
	if *randomDelay {
		rand.Seed(time.Now().UnixNano())
	}

	iterationsRun := 0
	var wg sync.WaitGroup
	screenChan := make(chan gocv.Mat, 1)

	for *iterationCount == 0 || iterationsRun < *iterationCount {
		iterationsRun++
		logChan <- fmt.Sprintf("Starting iteration %d", iterationsRun)

		isRunning, x, y, width, height, err := IsAppRunningAndLocateWindow("WhiteoutSurvival")
		if err != nil {
			logChan <- fmt.Sprintf("Error locating window: %v", err)
			continue
		}

		if isRunning {
			FocusApp("WhiteoutSurvival")
			time.Sleep(500 * time.Millisecond)

			wg.Add(1)
			go CaptureScreen(x, y, width, height, screenChan, &wg)

			wg.Wait()

			screen := <-screenChan
			if screen.Empty() {
				logChan <- "No valid screenshot received"
				continue
			}

			// Search and click big_x.png first
			if SearchAndClick(bigXTemplate, screen, x, y, width, height, "big_x") {
				// Wait for 0.25 seconds before capturing the next screenshot
				time.Sleep(250 * time.Millisecond)

				// Capture the screen again after clicking big_x.png
				wg.Add(1)
				go CaptureScreen(x, y, width, height, screenChan, &wg)
				wg.Wait()

				screen = <-screenChan
				if screen.Empty() {
					logChan <- "No valid screenshot received after big_x click"
					continue
				}
			}

			// Search and click handshake_icon.png
			SearchAndClick(handshakeTemplate, screen, x, y, width, height, "handshake_icon")

			screen.Close()
		} else {
			logChan <- "Application not running"
		}

		// Print iteration progress and help count
		if *iterationCount > 0 {
			logChan <- fmt.Sprintf("Completed iteration %d of %d", iterationsRun, *iterationCount)
		} else {
			logChan <- fmt.Sprintf("Completed iteration %d", iterationsRun)
		}

		// Use random delay if enabled, otherwise use fixed delay
		var delay time.Duration
		if *randomDelay {
			delay = time.Duration(rand.Intn(*iterationDelay)) * time.Second
		} else {
			delay = time.Duration(*iterationDelay) * time.Second
		}
		logChan <- fmt.Sprintf("Waiting for %v before next iteration", delay)
		time.Sleep(delay)
	}

	logChan <- fmt.Sprintf("Completed all %d iterations. Exiting.", iterationsRun)
}
