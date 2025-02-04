package main

import (
	"fmt"
	"image"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
	"gocv.io/x/gocv"
)

// Rect describes a rectangle with an origin (X, Y) and a size (Width, Height).
type Rect struct {
	X, Y, Width, Height int
}

var (
	logChan     = make(chan string, 100)
	mu          sync.Mutex
	totalClicks int
	appName	 = "whiteoutsurvival"
	// Create a local random generator instead of using rand.Seed globally.
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// logRoutine continuously logs messages with a timestamp.
func logRoutine() {
	for logMsg := range logChan {
		log.Printf("%s - %s", time.Now().Format("15:04:05"), logMsg)
	}
}

// adjustClickCoordinates converts a point (relative to the application window) into global screen coordinates.
func adjustClickCoordinates(pt image.Point, window Rect) image.Point {
	absX := window.X + pt.X
	absY := window.Y + pt.Y
	adjusted := image.Pt(absX, absY)
	logChan <- fmt.Sprintf("Translating in-window point %v with window offset (%d,%d) to desktop point %v",
		pt, window.X, window.Y, adjusted)
	return adjusted
}

// validateClickPoint checks if the absolute point is within the union of all active display bounds.
func validateClickPoint(pt image.Point) (bool, error) {
	union, err := getUnionDisplayBounds()
	if err != nil {
		return false, fmt.Errorf("unable to get union display bounds: %v", err)
	}
	if pt.X < union.X || pt.X > union.X+union.Width ||
		pt.Y < union.Y || pt.Y > union.Y+union.Height {
		return false, fmt.Errorf("click point %v is outside display bounds %+v", pt, union)
	}
	return true, nil
}

// clickAtAbsolutePoint calls the macOS-specific function to simulate a mouse click at the given point.
func clickAtAbsolutePoint(pt image.Point) {
	clickAtAbsolutePointCG(pt)
}

// incrementClicks safely increments the global totalClicks counter.
func incrementClicks() {
	mu.Lock()
	totalClicks++
	mu.Unlock()
}

// clickOnElement converts a relative click point into a global coordinate,
// clicks at that location (if the coordinate is valid),
// and then moves the mouse to a random coordinate inside the window.
func clickOnElement(matchPt image.Point, window Rect) {
	absPt := adjustClickCoordinates(matchPt, window)
	if valid, err := validateClickPoint(absPt); err != nil {
		logChan <- fmt.Sprintf("Warning: %v", err)
	} else if valid {
		clickAtAbsolutePoint(absPt)
		incrementClicks()
	}
	// Optional delays after clicking.
	time.Sleep(200 * time.Millisecond)
	// Move the mouse to a random coordinate within the window.
	moveMouseRandomInsideWindow(window)
}

// moveMouseRandomInsideWindow generates a random coordinate within the given window and moves the mouse there.
func moveMouseRandomInsideWindow(window Rect) {
	x := window.X + rng.Intn(window.Width)
	y := window.Y + rng.Intn(window.Height)
	randomPt := image.Pt(x, y)
	logChan <- fmt.Sprintf("Moving mouse to random coordinate inside window: %v", randomPt)
	moveMouseToCoordinateCG(randomPt)
}

// optimizedTemplateMatch performs multi-scale template matching on the screenshot using the given template.
// It returns the best matching location (as a point relative to the screenshot) and the corresponding match score.
func optimizedTemplateMatch(template gocv.Mat, screen gocv.Mat, targetName string) (image.Point, float32) {
	if template.Empty() || screen.Empty() {
		logChan <- fmt.Sprintf("Empty template or screen for %s", targetName)
		return image.Point{}, 0.0
	}

	// Convert both images to grayscale.
	grayScreen := gocv.NewMat()
	defer grayScreen.Close()
	gocv.CvtColor(screen, &grayScreen, gocv.ColorBGRToGray)

	grayTemplate := gocv.NewMat()
	defer grayTemplate.Close()
	gocv.CvtColor(template, &grayTemplate, gocv.ColorBGRToGray)

	var bestMatch float32
	var bestLoc image.Point

	// Try multiple scales to handle varying template sizes.
	for scale := 0.5; scale <= 2.0; scale += 0.1 {
		resized := gocv.NewMat()
		gocv.Resize(grayTemplate, &resized, image.Point{}, scale, scale, gocv.InterpolationLinear)
		if resized.Empty() {
			resized.Close()
			continue
		}

		result := gocv.NewMat()
		gocv.MatchTemplate(grayScreen, resized, &result, gocv.TmCcoeffNormed, gocv.NewMat())
		_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)
		if maxVal > bestMatch {
			bestMatch = maxVal
			bestLoc = maxLoc
		}
		resized.Close()
		result.Close()
	}

	logChan <- fmt.Sprintf("Best match for %s: %.2f at %v", targetName, bestMatch, bestLoc)
	return bestLoc, bestMatch
}

// detectAndClick loads a template image, performs template matching on the screenshot,
// applies fixed X and Y offsets to fine-tune the click location, and if the match score
// meets or exceeds the threshold, clicks on the element.
func detectAndClick(screen gocv.Mat, tmplPath string, window Rect, threshold float32, debug bool) bool {
	tmpl := gocv.IMRead(tmplPath, gocv.IMReadColor)
	if tmpl.Empty() {
		logChan <- fmt.Sprintf("Cannot open template: %s", tmplPath)
		return false
	}
	defer tmpl.Close()

	templateName := strings.TrimSuffix(filepath.Base(tmplPath), filepath.Ext(tmplPath))
	if debug {
		logChan <- fmt.Sprintf("Detecting [%s]...", templateName)
	}

	pt, bestMatch := optimizedTemplateMatch(tmpl, screen, templateName)

	// Apply fixed offsets: 3 pixels right and 6 pixels down.
	const xOffset = 3
	const yOffset = 6
	pt.X += xOffset
	pt.Y += yOffset

	if bestMatch >= threshold {
		logChan <- fmt.Sprintf("[%s] match: %.2f; clicking at relative position %v (with X-offset %d / Y-offset %d)",
			templateName, bestMatch, pt, xOffset, yOffset)
		clickOnElement(pt, window)
		return true
	}
	logChan <- fmt.Sprintf("[%s] not found (best match %.2f, threshold %.2f)", templateName, bestMatch, threshold)
	return false
}

// processWindow is the main routine that:
//   - Retrieves the application window geometry.
//   - Forces the application to come to the front.
//   - Captures a screenshot of the window.
//   - Runs template detection and clicking actions.
func processWindow(threshold float32, debug bool) {
	// Retrieve the window geometry via AppleScript.
	window, err := getAppWindowInfo()
	if err != nil {
		logChan <- fmt.Sprintf("Error retrieving window info: %v", err)
		return
	}

	// Bring the application to the front (and un-minimize if necessary).
	focusApp()
	time.Sleep(200 * time.Millisecond)

	// Capture the screenshot of the application's window.
	screen, err := captureScreen(window)
	if err != nil || screen.Empty() {
		logChan <- fmt.Sprintf("Failed to capture game window: %v", err)
		return
	}
	defer screen.Close()

	if debug {
		timestamp := time.Now().Format("20060102-150405")
		filename := fmt.Sprintf("debug_window_%s.png", timestamp)
		gocv.IMWrite(filename, screen)
		logChan <- fmt.Sprintf("Saved debug screenshot: %s", filename)
	}

	// Attempt to detect and click the "big_x" element.
	if detectAndClick(screen, "images/big_x.png", window, threshold, debug) {
		time.Sleep(1 * time.Second)
		newScreen, err := captureScreen(window)
		if err == nil && !newScreen.Empty() {
			if debug {
				timestamp := time.Now().Format("20060102-150405")
				filename := fmt.Sprintf("debug_after_x_%s.png", timestamp)
				gocv.IMWrite(filename, newScreen)
				logChan <- fmt.Sprintf("Saved debug screenshot: %s", filename)
			}
			detectAndClick(newScreen, "images/handshake_icon.png", window, threshold, debug)
			newScreen.Close()
		}
	} else {
		detectAndClick(screen, "images/handshake_icon.png", window, threshold, debug)
	}
}

func main() {
	app := &cli.App{
		Name:  "FrostyFriend",
		Usage: "Whiteout Survival automation helper (multi-monitor macOS version)",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "delay",
				Aliases: []string{"d"},
				Value:   5,
				Usage:   "Delay (in seconds) between iterations",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Value:   false,
				Usage:   "Enable verbose debug logging",
			},
			&cli.Float64Flag{
				Name:    "threshold",
				Aliases: []string{"t"},
				Value:   0.77,
				Usage:   "Matching threshold (0.0-1.0)",
			},
			&cli.IntFlag{
				Name:    "runs",
				Aliases: []string{"r"},
				Value:   0,
				Usage:   "Total number of runs (0 for infinite)",
			},
		},
		Action: func(c *cli.Context) error {
			delay := c.Int("delay")
			debug := c.Bool("debug")
			threshold := float32(c.Float64("threshold"))
			runLimit := c.Int("runs") // 0 means infinite runs

			go logRoutine()
			logChan <- "FrostyFriend started (multi-monitor macOS version)"

			var runCount int
			var prevClicks int

			// Main run loop.
			for {
				runCount++
				processWindow(threshold, debug)

				// Determine the status for this run:
				// If totalClicks increased since last iteration, at least one click (heal) occurred.
				var statusMark string
				if totalClicks > prevClicks {
					statusMark = "✔"
				} else {
					statusMark = "❌"
				}
				prevClicks = totalClicks

				// If debug is off, log a minimal summary for this run.
				if !debug {
					var runLimitStr string
					if runLimit > 0 {
						runLimitStr = fmt.Sprintf("%d", runLimit)
					} else {
						runLimitStr = "♾️ (infinite)"
					}
					log.Printf("Run %d/%s: %s (Heals: %d)", runCount, runLimitStr, statusMark, totalClicks)
				}

				// Exit if we've reached the run limit.
				if runLimit > 0 && runCount >= runLimit {
					break
				}
				time.Sleep(time.Duration(delay) * time.Second)
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}