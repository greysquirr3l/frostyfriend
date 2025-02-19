# FrostyFriend

FrostyFriend is a macOS-only automation helper for Whiteout Survival. It uses GoCV for computer vision and leverages AppleScript and Core Graphics for interacting with macOS applications.

## Features

- Brings the target application to the foreground.
- Captures screenshots of the application's window.
- Performs multi-scale template matching.
- Simulates mouse clicks and movements.

## Requirements

- macOS (this project is macOS-only)
- Go 1.16 or later
- [GoCV](https://gocv.io/)

## OS Dependencies

FrostyFriend uses macOS built-in tools:
- AppleScript for window control.
- The `screencapture` command for taking screenshots.
- Core Graphics (via the ApplicationServices framework) for simulating mouse events.

### Installing OpenCV (for GoCV)

Install OpenCV and pkg-config via Homebrew:
```
brew install opencv pkg-config
```

### Installing GoCV

Install GoCV using the following command:
```
go get -u -d gocv.io/x/gocv
```

## Usage

To run the script, use the following command:

```sh
go run main.go [options]
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
