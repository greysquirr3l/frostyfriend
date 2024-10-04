# 🎮 Whiteout Survival Helper 🤖

Automate some of Whiteout Survival gameplay with this nifty helper! 🚀

## 🌟 Features

- 🔍 Automatically detects and clicks on the handshake icon
- 🕰️ Customizable delay between iterations
- 🎲 Option for random delays to avoid detection
- 🔢 Set a specific number of iterations or run indefinitely
- 📊 Tracks successful "helps" (icon clicks)

## 🛠️ Prerequisites

- Go programming language installed on your system
- OpenCV for Go (gocv) library
- Whiteout Survival game installed on your Mac

## 🚀 Installation

1. Clone this repository:
   ```
   git clone https://github.com/yourusername/whiteout-survival-helper.git
   ```
2. Navigate to the project directory:
   ```
   cd whiteout-survival-helper
   ```
3. Install dependencies:
   ```
   go get -u gocv.io/x/gocv
   ```

## 🏃‍♂️ Usage

Set the WOS window to any screen that can see the Alliance Help popup.  If you are on a screen where the popup is not displayed, the script will still try to identify it, but it won't do anything because it won't find it.

Run the script with the following command:

```
go run main.go [flags]
```

### 🚩 Available Flags

- `-delay int`: Delay between iterations in seconds (default 10)
- `-random`: Use random delay between 0 and specified delay
- `-iterations int`: Number of iterations to run (0 for infinite)
- `-help`: Display help information

### 📘 Examples

Run with default settings:
```
go run main.go
```

Run with a 15-second delay and random timing:
```
go run main.go -delay 15 -random
```

Run for exactly 100 iterations:
```
go run main.go -iterations 100
```

## 📊 Output

The script will provide real-time feedback on its progress:

- 🏁 Start of each iteration
- 👆 When the handshake icon is clicked
- 🔢 Total number of successful helps
- ⏱️ Time until the next iteration

## ⚠️ Disclaimer

This script relies on AppleScript and screencapture from MacOS.  It was tested on macOS 10.15.

This script is for educational purposes only. Use it responsibly and at your own risk. Automating gameplay may violate the terms of service of some games.

## 🤝 Contributing

Feel free to fork this repository and submit pull requests. All contributions are welcome! 🎉

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

---

Happy gaming, and may your survival skills be ever sharp! ❄️🐧🏔️
