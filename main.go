package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sync"

	qrcode "github.com/skip2/go-qrcode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

//go:embed web.html
var webFS embed.FS

//go:embed logo.png
var iconBytes []byte // Embed the logo image

const (
	AppName = "PC Power Link"
	Version = "2.5"
	Port    = 8000
)

const LicenseText = `MIT License

Copyright (c) 2025 Nibil Krishna

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
SOFTWARE.`

// ================= STATE ==================

type State struct {
	mu           sync.RWMutex
	password     string
	device       string
	running      bool
	authRequired bool
	server       *http.Server
}

var state State
var webHTML []byte

func (s *State) SetPassword(v string) { s.mu.Lock(); s.password = v; s.mu.Unlock() }
func (s *State) Password() string     { s.mu.RLock(); defer s.mu.RUnlock(); return s.password }

func (s *State) SetDevice(v string) { s.mu.Lock(); s.device = v; s.mu.Unlock() }
func (s *State) Device() string     { s.mu.RLock(); defer s.mu.RUnlock(); return s.device }

func (s *State) SetRunning(v bool) { s.mu.Lock(); s.running = v; s.mu.Unlock() }
func (s *State) Running() bool     { s.mu.RLock(); defer s.mu.RUnlock(); return s.running }

func (s *State) SetAuthRequired(v bool) { s.mu.Lock(); s.authRequired = v; s.mu.Unlock() }
func (s *State) AuthRequired() bool     { s.mu.RLock(); defer s.mu.RUnlock(); return s.authRequired }

func (s *State) SetServer(v *http.Server) { s.mu.Lock(); s.server = v; s.mu.Unlock() }
func (s *State) Server() *http.Server     { s.mu.RLock(); defer s.mu.RUnlock(); return s.server }

// ================= HELPERS ==================

func primaryIP() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if ok && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "Unknown"
}

func baseURL() string {
	ip := primaryIP()
	if ip == "Unknown" {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", ip, Port)
}

// ================= POWER CMDS ==================

func doShutdown() {
	if runtime.GOOS == "windows" {
		exec.Command("shutdown", "/s", "/t", "0").Run()
	} else {
		exec.Command("systemctl", "poweroff").Run()
	}
}

func doRestart() {
	if runtime.GOOS == "windows" {
		exec.Command("shutdown", "/r", "/t", "0").Run()
	} else {
		exec.Command("systemctl", "reboot").Run()
	}
}

func doLock() {
	if runtime.GOOS == "windows" {
		exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
	} else {
		exec.Command("loginctl", "lock-session").Run()
	}
}

// ================= HTTP SERVER ==================

func authOK(r *http.Request) bool {
	if !state.AuthRequired() {
		return true
	}
	return r.Header.Get("X-Key") == state.Password()
}

func handleWeb(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(webHTML)
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"device":        state.Device(),
		"ip":            primaryIP(),
		"url":           baseURL(),
		"version":       Version,
		"auth_required": state.AuthRequired(),
	}
	json.NewEncoder(w).Encode(out)
}

func handleAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authOK(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		go func() {
			switch action {
			case "shutdown":
				doShutdown()
			case "restart":
				doRestart()
			case "lock":
				doLock()
			}
		}()
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

func startServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleWeb)
	mux.HandleFunc("/api/info", handleInfo)
	mux.HandleFunc("/api/power/shutdown", handleAction("shutdown"))
	mux.HandleFunc("/api/power/restart", handleAction("restart"))
	mux.HandleFunc("/api/power/lock", handleAction("lock"))

	srv := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", Port), Handler: mux}
	state.SetServer(srv)
	state.SetRunning(true)
	go srv.ListenAndServe()
}

func stopServer() {
	if state.Server() != nil {
		state.Server().Close()
	}
	state.SetRunning(false)
	state.SetServer(nil)
}

func makeQR(url string) image.Image {
	if url == "" {
		return nil
	}
	data, _ := qrcode.Encode(url, qrcode.Medium, 256)
	img, _ := png.Decode(bytes.NewReader(data))
	return img
}

// ================= MAIN UI ==================

func main() {
	myApp := app.NewWithID("com.nibilkrishna.pcpowerlink")
	
	// --- NEW: Set Application Icon ---
	// Fyne handles PNGs best for the running window icon.
	// If the file is missing during compile, "go build" will fail.
	iconRes := fyne.NewStaticResource("icon.png", iconBytes)
	myApp.SetIcon(iconRes)
	
	win := myApp.NewWindow(AppName)
	win.Resize(fyne.NewSize(450, 700))

	prefs := myApp.Preferences()
	savedDevice := prefs.StringWithFallback("deviceName", "")
	savedPassword := prefs.StringWithFallback("password", "")
	prefAutoStart := prefs.BoolWithFallback("autoStart", false)
	prefAuthRequired := prefs.BoolWithFallback("authRequired", true)

	if savedDevice == "" {
		savedDevice, _ = os.Hostname()
	}

	data, err := webFS.ReadFile("web.html")
	if err != nil {
		webHTML = []byte(fmt.Sprintf("<html><body><h1>Error: %v</h1></body></html>", err))
	} else {
		webHTML = data
	}

	// --- WIDGETS ---

	deviceEntry := widget.NewEntry()
	deviceEntry.SetText(savedDevice)
	
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter secret password")
	passwordEntry.SetText(savedPassword)
	
	authReqCheck := widget.NewCheck("Require Password", func(checked bool) {
		if checked {
			passwordEntry.Enable()
		} else {
			passwordEntry.Disable()
		}
		prefs.SetBool("authRequired", checked)
	})
	authReqCheck.SetChecked(prefAuthRequired)
	if !prefAuthRequired { passwordEntry.Disable() }

	autoStartCheck := widget.NewCheck("Auto-start server on app open", func(checked bool) {
		prefs.SetBool("autoStart", checked)
	})
	autoStartCheck.SetChecked(prefAutoStart)

	statusIcon := widget.NewIcon(theme.MediaStopIcon())
	statusLabel := widget.NewLabel("Stopped")
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}
	
	ipEntry := widget.NewEntry()
	ipEntry.SetPlaceHolder("IP Address")
	ipEntry.Disable() 
	
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Connection URL")
	urlEntry.Disable()

	qrImg := canvas.NewImageFromImage(nil)
	qrImg.FillMode = canvas.ImageFillContain
	qrImg.SetMinSize(fyne.NewSize(200, 200))

	copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
		if urlEntry.Text != "" {
			win.Clipboard().SetContent(urlEntry.Text)
		}
	})

	var startBtn *widget.Button
	
	toggleServer := func() {
		if !state.Running() {
			// START
			if authReqCheck.Checked && passwordEntry.Text == "" {
				dialog.ShowInformation("Error", "Please set a password in Settings", win)
				return
			}
			
			prefs.SetString("deviceName", deviceEntry.Text)
			prefs.SetString("password", passwordEntry.Text)
			
			state.SetPassword(passwordEntry.Text)
			state.SetDevice(deviceEntry.Text)
			state.SetAuthRequired(authReqCheck.Checked)
			
			startServer()
			
			currentIP := primaryIP()
			currentURL := baseURL()
			
			statusIcon.SetResource(theme.MediaPlayIcon())
			statusLabel.SetText("Running")
			statusLabel.TextStyle = fyne.TextStyle{Bold: true, Italic: false}
			
			ipEntry.SetText(currentIP)
			urlEntry.SetText(currentURL)
			qrImg.Image = makeQR(currentURL)
			qrImg.Refresh()
			
			startBtn.SetText("Stop Server")
			startBtn.Importance = widget.DangerImportance
			startBtn.Icon = theme.MediaStopIcon()
			
			deviceEntry.Disable()
			passwordEntry.Disable()
			authReqCheck.Disable()
			
		} else {
			// STOP
			stopServer()
			
			statusIcon.SetResource(theme.MediaStopIcon())
			statusLabel.SetText("Stopped")
			
			ipEntry.SetText("")
			urlEntry.SetText("")
			qrImg.Image = nil
			qrImg.Refresh()
			
			startBtn.SetText("Start Server")
			startBtn.Importance = widget.HighImportance
			startBtn.Icon = theme.MediaPlayIcon()
			
			deviceEntry.Enable()
			authReqCheck.Enable()
			if authReqCheck.Checked {
				passwordEntry.Enable()
			}
		}
	}

	startBtn = widget.NewButtonWithIcon("Start Server", theme.MediaPlayIcon(), toggleServer)
	startBtn.Importance = widget.HighImportance

	// --- LAYOUT ---

	// TAB 1: DASHBOARD
	statusContent := container.NewVBox(
		container.NewHBox(statusIcon, statusLabel),
		widget.NewSeparator(),
		widget.NewLabel("Local IP"),
		ipEntry,
		widget.NewLabel("Connection URL"),
		container.NewBorder(nil, nil, nil, copyBtn, urlEntry),
	)
	statusCard := widget.NewCard("Status", "", statusContent)

	qrContent := container.NewCenter(qrImg)
	qrCard := widget.NewCard("Mobile Connect", "Scan to control this PC", qrContent)

	dashboardLayout := container.NewVBox(
		statusCard,
		widget.NewSeparator(),
		qrCard,
		layout.NewSpacer(),
		startBtn,
	)

	// TAB 2: SETTINGS
	identCard := widget.NewCard("Identification", "How this PC appears", container.NewVBox(
		widget.NewLabel("Device Name"),
		deviceEntry,
		widget.NewLabel("Password"),
		passwordEntry,
	))

	behaviorCard := widget.NewCard("Behavior", "App configuration", container.NewVBox(
		authReqCheck,
		autoStartCheck,
	))

	settingsLayout := container.NewVScroll(container.NewVBox(
		identCard,
		behaviorCard,
	))

	// TAB 3: ABOUT
	githubURL, _ := url.Parse("https://github.com/nibilXD/pc-power-link")
	
	titleLabel := widget.NewLabel(AppName)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter
	
	verLabel := widget.NewLabel("Version " + Version)
	verLabel.Alignment = fyne.TextAlignCenter
	
	copyLabel := widget.NewLabel("Copyright (c) 2025 Nibil Krishna")
	copyLabel.Alignment = fyne.TextAlignCenter
	
	ghLink := widget.NewHyperlink("View on GitHub", githubURL)
	ghLink.Alignment = fyne.TextAlignCenter

	licenseEntry := widget.NewMultiLineEntry()
	licenseEntry.SetText(LicenseText)
	licenseEntry.Disable()
	licenseEntry.Wrapping = fyne.TextWrapWord
	
	licenseContainer := container.NewGridWrap(fyne.NewSize(400, 300), licenseEntry)

	licenseHeader := widget.NewLabel("License")
	licenseHeader.TextStyle = fyne.TextStyle{Bold: true}
	licenseHeader.Alignment = fyne.TextAlignCenter

	aboutContent := container.NewVBox(
		layout.NewSpacer(),
		// Use the embedded icon for the About page logo as well
		widget.NewIcon(iconRes),
		titleLabel,
		verLabel,
		ghLink,
		layout.NewSpacer(),
		copyLabel,
		licenseHeader,
		licenseContainer,
	)

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Dashboard", theme.HomeIcon(), container.NewPadded(dashboardLayout)),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), container.NewPadded(settingsLayout)),
		container.NewTabItemWithIcon("About", theme.InfoIcon(), container.NewPadded(container.NewVScroll(aboutContent))),
	)

	win.SetContent(tabs)

	if prefAutoStart {
		if !prefAuthRequired || (prefAuthRequired && savedPassword != "") {
			toggleServer()
		}
	}

	win.ShowAndRun()
}
