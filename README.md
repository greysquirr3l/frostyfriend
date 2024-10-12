# Whiteout Survival Helper

This script automates interactions with the game "Whiteout Survival." It locates the game's window, performs image matching to identify specific in-game elements, and automates mouse clicks to assist with gameplay.

## Features
- Detects the presence of two specific images (`big_x.png` and `handshake_icon.png`) within the game window.
- Automatically clicks on detected images based on their priority:
  - If `big_x.png` is detected, it clicks to exit out of an unintended screen (e.g., chat window).
  - It then waits briefly, takes another screenshot, and attempts to click on `handshake_icon.png`.
- Annotates detected images with debug markers if `--debug` flag is enabled.
- Supports infinite or specified number of iterations with configurable delay between iterations.
- Uses `cliclick` to perform automated mouse clicks.

## Prerequisites
- macOS system (uses AppleScript and `osascript` to locate and interact with the game window).
- GoCV library to perform image processing and template matching.
- `cliclick` tool for command-line mouse click automation.
- Game window must be visible for interactions to succeed.

## Installation
1. Ensure you have Go installed on your system.
2. Install the GoCV package:
   ```sh
   go get -u -d gocv.io/x/gocv
   ```
3. Install `cliclick` for simulating mouse clicks:
   ```sh
   brew install cliclick
   ```
4. Download or create the necessary image templates (`big_x.png` and `handshake_icon.png`) and store them in an `images` directory in the same path as the script.

## Usage
To run the script, use the following command:

<<<<<<< HEAD
```sh
go run main.go [options]
=======
Set the WOS window to any screen that can see the Alliance Help popup.  If you are on a screen where the popup is not displayed, the script will still try to identify it, but it won't do anything because it won't find it.

Run the script with the following command:

```
go run main.go [flags]
>>>>>>> main
```

### Command-Line Options
- `--iterations`: Number of iterations to run the helper. Set to `0` for infinite iterations. (default: 0)
- `--delay`: Delay between iterations, in seconds. (default: 10)
- `--random`: If set, use a random delay between `0` and the specified delay.
- `--debug`: Enable debug mode to save screenshots annotated with rectangles and click points.
- `--help`: Display usage information.

### Example Commands
Run with a fixed delay of 5 seconds between iterations and infinite runs:
```sh
go run main.go --delay 5
```
Run for 10 iterations with random delay up to 8 seconds between iterations:
```sh
go run main.go --iterations 10 --delay 8 --random
```
Run in debug mode to annotate and save screenshots:
```sh
go run main.go --debug
```

## How It Works
1. **Application Detection and Window Location**: The script checks if "WhiteoutSurvival" is running and locates its window.
2. **Screen Capture**: It captures the game window to identify target images.
3. **Image Detection and Interaction**:
   - Searches for `big_x.png`. If found, it clicks on the location and waits for a brief delay.
   - Takes a new screenshot, searches for `handshake_icon.png`, and clicks on it if found.
4. **Debug Mode**: When enabled, the script saves annotated screenshots to help visualize the matched areas and click locations.
5. **Logging**: Logs significant events (e.g., image matches, clicks, errors) for easier tracking.

## Dependencies
- **GoCV**: Handles image processing and template matching.
- **AppleScript (`osascript`)**: Locates and interacts with the application window.
- **`cliclick`**: Performs the automated mouse clicks.

## Important Notes
- The script currently only works on macOS due to its reliance on `osascript` and the `screencapture` command.
- Ensure the game window remains visible and unobstructed during the script's runtime to ensure proper functionality.
- Make sure the images used for template matching (`big_x.png` and `handshake_icon.png`) closely resemble the in-game visuals to ensure accurate detection.

## License
This script is provided "as is" without warranty of any kind. Use it at your own risk.

## Author
Nick Campbell

## Contributions
Feel free to open issues or contribute enhancements to the script. Suggestions and improvements are welcome.
