# ⚡ PC Power Link

> **Control your PC's power state remotely from any device on your local network.**

![Version](https://img.shields.io/badge/version-2.5-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green.svg)

**PC Power Link** is a lightweight, cross-platform desktop application that turns your computer into a local server. It generates a QR code that you can scan with your smartphone to instantly access a secure web interface to **Shutdown**, **Restart**, or **Lock** your PC.

---



## ✨ Features

* **📱 Instant Connection:** Generates a QR code for quick mobile access.
* **🔒 Secure:** Optional password protection to prevent unauthorized access.
* **🔌 Power Controls:** Remote Shutdown, Restart, and Lock.
* **🌗 Dark Mode:** The web interface automatically adapts to your system theme or can be toggled manually.
* **🐧 Cross-Platform:** Works seamlessly on **Linux** (systemd) and **Windows**.
* **⚙️ User Settings:** Configurable device name, auto-start options, and security settings.
* **📦 Single Binary:** The web interface is embedded into the app—no extra files needed to run.

---

## 🛠️ Tech Stack

* **Backend & GUI:** [Go](https://go.dev/) + [Fyne](https://fyne.io/)
* **Frontend:** HTML5 + [TailwindCSS](https://tailwindcss.com/) (CDN)
* **Network:** Standard Go `net/http` library

---

## 🚀 Installation & Build

### Prerequisites
* [Go](https://go.dev/dl/) installed (version 1.21 or higher recommended).
* **Linux Users:** You need C compiler and graphics headers for Fyne:
    ```bash
    sudo apt-get install golang gcc libgl1-mesa-dev xorg-dev
    ```

### Building from Source

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/nibilXD/pc-power-link.git](https://github.com/nibilXD/pc-power-link.git)
    cd pc-power-link
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Build the app:**
    ```bash
    # Linux
    go build -o PCPowerLink main.go

    # Windows (Hides the console window)
    go build -ldflags "-H=windowsgui" -o "PC Power Link.exe" main.go
    ```

---

## 🐧 Linux Desktop Integration

To add the app to your Linux application menu:

1.  Move the built binary and `logo.png` to a permanent folder (e.g., `~/pc-power-link/`).
2.  Create a desktop entry:
    ```bash
    nano ~/.local/share/applications/pc-power-link.desktop
    ```
3.  Paste the following (update paths to match your location):
    ```ini
    [Desktop Entry]
    Type=Application
    Name=PC Power Link
    Comment=Remote Power Control
    Exec=/home/YOUR_USER/pc-power-link/PCPowerLink
    Icon=/home/YOUR_USER/pc-power-link/logo.png
    Terminal=false
    Categories=Utility;System;
    ```

---

## 📖 Usage

1.  **Start the App:** Run the application on your computer.
2.  **Set a Password:** (Optional) Go to the **Settings** tab to set a password and enable "Require Password".
3.  **Start Server:** Click the "Start Server" button on the Dashboard.
4.  **Scan & Control:** Scan the QR code with your phone. If a password is set, enter it on your phone to unlock the controls.

---

## 📄 License

**MIT License**

Copyright (c) 2025 **Nibil Krishna**

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
