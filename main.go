package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand"
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

var clickTarget = ClickTarget{
	Name: "Handshake",
	X:    1.40,
	Y:    1.78,
}

var debugMode bool

func IsAppRunning(appName string) bool {
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

func LocateWindow(appName string) (int, int, int, int, error) {
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
		return 0, 0, 0, 0, fmt.Errorf("could not locate window: %v", err)
	}

	dimensions := strings.TrimSpace(string(out))
	dimensionSlice := strings.FieldsFunc(dimensions, func(r rune) bool {
		return r == ',' || r == ' '
	})

	if len(dimensionSlice) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("unexpected number of dimensions: %v", dimensionSlice)
	}

	x, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[0]))
	y, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[1]))
	width, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[2]))
	height, _ := strconv.Atoi(strings.TrimSpace(dimensionSlice[3]))

	return x, y, width, height, nil
}

func FocusApp(appName string) {
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to activate`, appName))
	if err := cmd.Run(); err != nil {
		log.Printf("Error focusing application %s: %v", appName, err)
	}
}

func CaptureScreen(windowX, windowY, width, height int) gocv.Mat {
	outputPath := "/tmp/screenshot.png"
	cmd := exec.Command("screencapture", "-R", fmt.Sprintf("%d,%d,%d,%d", windowX, windowY, width, height), outputPath)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting screen capture: %v", err)
		return gocv.NewMat()
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Failed to kill screen capture process: %v", err)
		}
		log.Println("Screen capture timed out")
		return gocv.NewMat()
	case err := <-done:
		if err != nil {
			log.Printf("Error capturing screen: %v", err)
			return gocv.NewMat()
		}
	}

	img := gocv.IMRead(outputPath, gocv.IMReadColor)
	if img.Empty() {
		log.Println("Failed to load screenshot")
		return gocv.NewMat()
	}

	return img
}

func SearchAndClick(template gocv.Mat, screen gocv.Mat, windowX, windowY, windowWidth, windowHeight int) bool {
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
			// Draw green rectangle around the matched region
			rect := image.Rect(maxLoc.X, maxLoc.Y, maxLoc.X+template.Cols(), maxLoc.Y+template.Rows())
			gocv.Rectangle(&screen, rect, color.RGBA{0, 255, 0, 0}, 2)

			// Draw red circle where the click will take place
			gocv.Circle(&screen, image.Pt(centerX, centerY), 5, color.RGBA{255, 0, 0, 0}, -1)

			// Save the debug screenshot
			debugPath := fmt.Sprintf("%s-debug.png", clickTarget.Name)
			if ok := gocv.IMWrite(debugPath, screen); !ok {
				log.Printf("Error saving debug screenshot: %s", debugPath)
			} else {
				log.Printf("Debug screenshot saved: %s", debugPath)
			}
		}

		cmd := exec.Command("cliclick", fmt.Sprintf("c:%d,%d", adjustedX, adjustedY))
		if err := cmd.Run(); err != nil {
			log.Printf("Error performing click: %v", err)
			return false
		}
		log.Printf("Click performed at position: (%d, %d)", adjustedX, adjustedY)
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
	log.Println("Starting Whiteout Survival helper")

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChannel
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	appName := "WhiteoutSurvival"

	templatePath := filepath.Join("images", "handshake_icon.png")
	template := gocv.IMRead(templatePath, gocv.IMReadColor)
	if template.Empty() {
		log.Fatalf("Error reading template image file: %s", templatePath)
	}
	defer template.Close()

	log.Printf("Template image loaded successfully from: %s", templatePath)

	// Initialize random number generator if random delay is enabled
	if *randomDelay {
		rand.Seed(time.Now().UnixNano())
	}

	iterationsRun := 0
	helpsCount := 0
	for *iterationCount == 0 || iterationsRun < *iterationCount {
		iterationsRun++
		log.Printf("Starting iteration %d", iterationsRun)

		if IsAppRunning(appName) {
			FocusApp(appName)
			time.Sleep(500 * time.Millisecond)

			x, y, width, height, err := LocateWindow(appName)
			if err != nil {
				log.Printf("Error locating window: %v", err)
				continue
			}

			screen := CaptureScreen(x, y, width, height)
			if screen.Empty() {
				log.Println("Failed to capture screen")
				continue
			}

			matched := SearchAndClick(template, screen, x, y, width, height)

			if matched {
				helpsCount++
				log.Printf("Handshake icon found and clicked (Total helps: %d)", helpsCount)
			} else {
				log.Println("Handshake icon not found in this iteration")
			}

			screen.Close()
		} else {
			log.Println("Application not running")
		}

		// Print iteration progress and help count
		if *iterationCount > 0 {
			log.Printf("Completed iteration %d of %d. Total helps: %d", iterationsRun, *iterationCount, helpsCount)
		} else {
			log.Printf("Completed iteration %d. Total helps: %d", iterationsRun, helpsCount)
		}

		// Use random delay if enabled, otherwise use fixed delay
		if *randomDelay {
			delay := time.Duration(rand.Intn(*iterationDelay)) * time.Second
			log.Printf("Waiting for %v before next iteration", delay)
			time.Sleep(delay)
		} else {
			delay := time.Duration(*iterationDelay) * time.Second
			log.Printf("Waiting for %v before next iteration", delay)
			time.Sleep(delay)
		}
	}

	log.Printf("Completed all %d iterations. Total helps: %d. Exiting.", iterationsRun, helpsCount)
}
