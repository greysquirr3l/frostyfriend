package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"myproject/locate"

	"gocv.io/x/gocv"
)

// LoadImages loads all PNG images from the images folder into a matrix
func LoadImages(folderPath string) (map[string]gocv.Mat, error) {
	images := make(map[string]gocv.Mat)

	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".png" {
			img := gocv.IMRead(path, gocv.IMReadColor)
			if img.Empty() {
				return fmt.Errorf("could not read image: %s", path)
			}
			log.Printf("Loaded image: %s", info.Name())
			images[info.Name()] = img
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return images, nil
}

func main() {
	// Setup logging to a file
	logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println("Error setting up log file:", err)
		return
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	log.Println("Starting Whiteout Survival helper")

	// Handle CTRL-C (SIGINT) gracefully
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChannel
		log.Println("Shutting down...")
		os.Exit(0)
	}()

	// Load images into a map
	images, err := LoadImages("./images")
	if err != nil {
		log.Fatalf("Failed to load images: %v", err)
	}

	// Check if Whiteout Survival is running and monitor it every second
	appName := "Whiteout Survival"
	for {
		if locate.IsAppRunning(appName) {
			x, y, width, height, err := locate.LocateWindow(appName)
			if err != nil {
				log.Printf("Error locating window: %v", err)
			} else {
				log.Printf("Monitoring window at position: (%d, %d), size: (%d, %d)", x, y, width, height)

				// Capture the screen (replace with actual screen capture logic)
				screen := gocv.IMRead("path_to_screen_capture", gocv.IMReadColor) // Dummy placeholder
				if screen.Empty() {
					log.Println("Failed to capture screen")
					continue
				}

				// Search for the help image and click it if found
				if helpImage, found := images["click_help.png"]; found {
					clicked := SearchAndClick(helpImage, screen)
					if clicked {
						log.Println("Click performed on click_help.png")
					}
				}
			}
		} else {
			log.Println("Application not running")
		}

		time.Sleep(1 * time.Second)
	}
}

// SearchAndClick searches for an image and clicks if found
func SearchAndClick(image gocv.Mat, screen gocv.Mat) bool {
	result := gocv.NewMat()
	gocv.MatchTemplate(screen, image, &result, gocv.TmCcoeffNormed)

	_, maxVal, _, maxLoc := gocv.MinMaxLoc(result)

	if maxVal >= 0.9 {
		log.Printf("Image found with match value: %.2f at position: (%d, %d)", maxVal, maxLoc.X, maxLoc.Y)
		// Simulate click (you can replace this with your actual click simulation)
		exec.Command("cliclick", fmt.Sprintf("c:%d,%d", maxLoc.X, maxLoc.Y)).Run()
		return true
	}

	log.Println("Image not found")
	return false
}
