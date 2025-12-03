# ⚡️ PC Power Link

A cross-platform desktop application and web server utility built with Go and Fyne that allows remote control (Lock, Restart, Shutdown) of your PC via any web browser on your network.

## ✨ Features

* **Remote Control:** Lock, Restart, and Shutdown commands via a simple web interface.
* **Secure:** Requires a password (set in the desktop app) for all commands.
* **Easy Setup:** Uses a QR code for quick connection from mobile devices.
* **Cross-Platform:** Works on Windows, macOS, and Linux (requires Fyne dependencies).

## 🚀 Getting Started (Building from Source)

This project requires **Go 1.16 or newer** (for the `embed` package).

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/YourUsername/pc-power-link.git](https://github.com/YourUsername/pc-power-link.git)
    cd pc-power-link
    ```

2.  **Install dependencies and build:**
    The application uses the `fyne` GUI library. Ensure you have the necessary system dependencies installed (see Fyne documentation for details).
    
    The build process embeds the necessary `web.html` file, resulting in a single executable.
    ```bash
    go mod tidy
    go build -o pc-power-link .
    ```

3.  **Run the application:**
    ```bash
    ./pc-power-link
    ```
    *Note: On Windows, the executable will be named `pc-power-link.exe`.*

## ⚙️ Usage

1.  Open the desktop application.
2.  Enter a strong password and click **"Start Server"**.
3.  Scan the QR code displayed with your phone or navigate to the URL shown (e.g., `http://192.168.1.5:8000`).
4.  Enter the password in the web page to gain access to the control buttons.

