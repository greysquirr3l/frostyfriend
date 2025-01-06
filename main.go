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

	"gocv.io/x/gocv"
)

// Global variables
var (
	debugMode   bool
	logChan     = make(chan string, 100)
	totalClicks int
)

func logRoutine() {
	for logMsg := range logChan {
		log.Printf("%s - %s", time.Now().Format("15:04:05"), logMsg)
	}
}

func IsAppRunningAndLocateWindow(appName string) (bool, int, int, int, int, error) {
	script := fmt.Sprintf(`
        tell application "System Events"
            set appList to (name of every process where background only is false)
            if "%s" is in appList then
                tell process "%s"
                    try
                        set appWindow to first window
                        set {x, y} to position of appWindow
                        set {width, height} to size of appWindow
                        return "true," & x & "," & y & "," & width & "," & height
                    on error errMsg
                        return "false,error," & errMsg
                    end try
                end tell
            else
                return "false,app_not_running"
            end if
        end tell
    `, appName, appName)

	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return false, 0, 0, 0, 0, fmt.Errorf("AppleScript execution failed: %v", err)
	}

	output := strings.TrimSpace(string(out))
	if strings.HasPrefix(output, "false") {
		parts := strings.Split(output, ",")
		if len(parts) > 1 && parts[1] == "error" {
			return false, 0, 0, 0, 0, fmt.Errorf("AppleScript error: %s", strings.Join(parts[2:], ","))
		}
		return false, 0, 0, 0, 0, nil
	}

	parts := strings.Split(output, ",")
	if len(parts) != 5 {
		return false, 0, 0, 0, 0, fmt.Errorf("unexpected output format: %s", output)
	}

	x, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	y, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	width, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	height, _ := strconv.Atoi(strings.TrimSpace(parts[4]))

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

	if width <= 0 || height <= 0 {
		logChan <- fmt.Sprintf("Invalid dimensions: width=%d, height=%d", width, height)
		screenChan <- gocv.NewMat()
		return
	}

	cmd := exec.Command("screencapture", "-x", "-R", fmt.Sprintf("%d,%d,%d,%d", windowX, windowY, width, height), outputPath)
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

func OptimizedSearchAndClick(template gocv.Mat, screen gocv.Mat, windowX, windowY, windowWidth, windowHeight int, targetName string) bool {
	if template.Empty() || screen.Empty() {
		logChan <- fmt.Sprintf("Empty template or screen for %s", targetName)
		return false
	}

	if screen.Channels() > 1 {
		gocv.CvtColor(screen, &screen, gocv.ColorBGRToGray)
	}
	if template.Channels() > 1 {
		gocv.CvtColor(template, &template, gocv.ColorBGRToGray)
	}

	var bestMatch float32
	var bestLocation image.Point
	for scale := 0.5; scale <= 2.0; scale += 0.1 {
		resized := gocv.NewMat()
		gocv.Resize(template, &resized, image.Point{}, scale, scale, gocv.InterpolationLinear)

		result := gocv.NewMat()
		gocv.MatchTemplate(screen, resized, &result, gocv.TmCcoeffNormed, gocv.NewMat())

		_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)
		if float32(maxVal) > bestMatch {
			bestMatch = float32(maxVal)
			bestLocation = maxLoc
		}
		resized.Close()
		result.Close()
	}

	threshold := 0.80
	if targetName == "big_x" {
		threshold = 0.90
	}

	if bestMatch >= float32(threshold) {
		centerX := bestLocation.X + (template.Cols() / 2)
		centerY := bestLocation.Y + (template.Rows() / 2)

		logChan <- fmt.Sprintf("Center coordinates before scaling: (%d, %d)", centerX, centerY)

		scaleX := float64(windowWidth) / float64(screen.Cols())
		scaleY := float64(windowHeight) / float64(screen.Rows())

		logChan <- fmt.Sprintf("Scaling factors: scaleX=%.2f, scaleY=%.2f", scaleX, scaleY)

		adjustedX := windowX + int(float64(centerX)*scaleX)
		adjustedY := windowY + int(float64(centerY)*scaleY)

		logChan <- fmt.Sprintf("Adjusted click coordinates: (%d, %d)", adjustedX, adjustedY)
		logChan <- fmt.Sprintf("Screen dimensions: cols=%d, rows=%d", screen.Cols(), screen.Rows())

		if adjustedX < 0 || adjustedY < 0 || adjustedX > windowX+windowWidth || adjustedY > windowY+windowHeight {
			logChan <- fmt.Sprintf("Invalid click coordinates: (%d, %d) for %s", adjustedX, adjustedY, targetName)
			return false
		}

		time.Sleep(100 * time.Millisecond)

		// Perform the click
		cmd := exec.Command("cliclick", fmt.Sprintf("c:%d,%d", adjustedX, adjustedY))
		logChan <- fmt.Sprintf("Executing cliclick command: c:%d,%d", adjustedX, adjustedY)
		if err := cmd.Run(); err != nil {
			logChan <- fmt.Sprintf("Error performing click: %v", err)
			return false
		}

		totalClicks++
		logChan <- fmt.Sprintf("Click performed on %s at (%d, %d) (Total clicks: %d, Confidence: %.2f)",
			targetName, adjustedX, adjustedY, totalClicks, bestMatch)

		// Move cursor to a random position inside the application window
		randomX := windowX + rand.Intn(windowWidth)
		randomY := windowY + rand.Intn(windowHeight)
		cmd = exec.Command("cliclick", fmt.Sprintf("m:%d,%d", randomX, randomY))
		logChan <- fmt.Sprintf("Moving cursor to random position inside the window: (%d, %d)", randomX, randomY)
		if err := cmd.Run(); err != nil {
			logChan <- fmt.Sprintf("Error moving cursor: %v", err)
			return false
		}

		return true
	}

	logChan <- fmt.Sprintf("No match found for %s: Best Confidence=%.2f", targetName, bestMatch)
	return false
}

func main() {
	iterationDelay := flag.Int("delay", 10, "Delay between iterations in seconds")
	randomDelay := flag.Bool("random", false, "Use random delay")
	iterationCount := flag.Int("iterations", 0, "Number of iterations (0 for infinite)")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode")
	flag.Parse()

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

	handshakeTemplate := gocv.IMRead(filepath.Join("images", "handshake_icon.png"), gocv.IMReadColor)
	defer handshakeTemplate.Close()

	bigXTemplate := gocv.IMRead(filepath.Join("images", "big_x.png"), gocv.IMReadColor)
	defer bigXTemplate.Close()

	screenChan := make(chan gocv.Mat, 1)
	var wg sync.WaitGroup

	for *iterationCount == 0 || totalClicks < *iterationCount {
		isRunning, x, y, width, height, err := IsAppRunningAndLocateWindow("whiteoutsurvival")
		if err != nil || !isRunning {
			logChan <- fmt.Sprintf("Application not running or error: %v", err)
			time.Sleep(time.Duration(*iterationDelay) * time.Second)
			continue
		}

		FocusApp("WhiteoutSurvival")
		wg.Add(1)
		go CaptureScreen(x, y, width, height, screenChan, &wg)
		wg.Wait()

		screen := <-screenChan
		if screen.Empty() {
			logChan <- "No valid screenshot received"
			continue
		}

		OptimizedSearchAndClick(bigXTemplate, screen, x, y, width, height, "big_x")
		OptimizedSearchAndClick(handshakeTemplate, screen, x, y, width, height, "handshake_icon")

		screen.Close()

		delay := time.Duration(*iterationDelay) * time.Second
		if *randomDelay {
			delay = time.Duration(rand.Intn(*iterationDelay)) * time.Second
		}
		logChan <- fmt.Sprintf("Waiting for %v before next iteration", delay)
		time.Sleep(delay)
	}
}
