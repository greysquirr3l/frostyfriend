# Whiteout Survival Helper 🎮✨

## Overview

The Whiteout Survival Helper is a Go-based automation tool designed to interact with the "Whiteout Survival" game. This application monitors the game for specific UI elements (like the handshake icon) and performs automated clicks based on the defined targets.

## Features 🌟

- **Monitor Application**: Continuously checks if the "Whiteout Survival" application is running.
- **Template Matching**: Uses OpenCV to locate predefined images (e.g., handshake icon) on the game window.
- **Automatic Clicks**: Simulates mouse clicks on the identified UI elements.
- **Logging**: Captures logs of actions taken and errors encountered.
- **Customizable Targets**: Easily add more target templates for additional interactions.
- **Randomized Delays**: Implements random delays to mimic human behavior.

## Prerequisites 🔧

Before you start, ensure you have the following installed:

- **Go**: [Download and install Go](https://golang.org/doc/install)
- **OpenCV**: Ensure you have GoCV installed. You can find installation instructions [here](https://gocv.io/getting-started/).
- **cliclick**: This tool is used to simulate mouse clicks. Install it via Homebrew:

    ```bash
    brew install cliclick
    ```

- **lumberjack**: For log file management. It is included as a dependency in the code.

## Installation ⚙️

1. Clone the repository:

    ```bash
    git clone https://github.com/yourusername/frostyfriend.git
    cd frostyfriend
    ```

2. Install the required Go packages:

    ```bash
    go get -u gocv.io/x/gocv
    go get gopkg.in/natefinch/lumberjack.v2
    ```

## Usage 🚀

1. **Add Target Images**: Place your target images (e.g., `handshake_icon.png`) in the `images` directory.

2. **Run the Application**:

    ```bash
    go run main.go
    ```

3. **Command-Line Flags**:
   - Add `--autoclose` if you want the application to close and reopen automatically on crash.

## Example of Command

```bash
go run main.go --autoclose
```
