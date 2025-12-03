package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"embed" // NEW: Go's built-in file embedding package

	qrcode "github.com/skip2/go-qrcode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// The //go:embed directive tells the Go compiler to embed the content
// of web.html into the webFS variable.
//go:embed web.html
var webFS embed.FS

const (
	AppName = "PC Power Link"
	Version = "1.0.0"
	Port    = 8000
)

// ================= STATE ==================

type State struct {
	mu       sync.RWMutex
	password string
	device   string
	running  bool
	server   *http.Server
}

var state State
var webHTML []byte // Holds the actual byte content read from webFS at startup

func (s *State) SetPassword(v string) { s.mu.Lock(); s.password = v; s.mu.Unlock() }
func (s *State) Password() string     { s.mu.RLock(); defer s.mu.RUnlock(); return s.password }

func (s *State) SetDevice(v string) { s.mu.Lock(); s.device = v; s.mu.Unlock() }
func (s *State) Device() string     { s.mu.RLock(); defer s.mu.RUnlock(); return s.device }

func (s *State) SetRunning(v bool) { s.mu.Lock(); s.running = v; s.mu.Unlock() }
func (s *State) Running() bool     { s.mu.RLock(); defer s.mu.RUnlock(); return s.running }

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
	return ""
}

func baseURL() string {
	ip := primaryIP()
	if ip == "" {
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
	return r.Header.Get("X-Key") == state.Password()
}

// handleWeb now serves the content loaded from the embedded file system
func handleWeb(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(webHTML)
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"device":  state.Device(),
		"ip":      primaryIP(),
		"url":     baseURL(),
		"version": Version,
	}
	json.NewEncoder(w).Encode(out)
}

func handleAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authOK(r) {
			// This returns the 401 status code which the client side now handles correctly
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

// ================= QR ==================

func makeQR(url string) image.Image {
	if url == "" {
		return nil
	}
	data, _ := qrcode.Encode(url, qrcode.Medium, 256)
	img, _ := png.Decode(bytes.NewReader(data))
	return img
}

// ================= UI ==================

func main() {
	host, _ := os.Hostname()
	state.SetDevice(host)
	
	// NEW: Load the content from the embedded webFS
	data, err := webFS.ReadFile("web.html")
	if err != nil {
		fmt.Printf("Error reading embedded web.html: %v\n", err)
		// Fallback content if the file can't be read from the embedFS
		webHTML = []byte(fmt.Sprintf("<html><body><h1>Fatal Error: Could not load embedded web page (%v)</h1></body></html>", err))
	} else {
		webHTML = data
	}

	myApp := app.New()
	win := myApp.NewWindow(AppName)
	win.Resize(fyne.NewSize(800, 600))

	deviceEntry := widget.NewEntry()
	deviceEntry.SetText(state.Device())

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Enter password")

	status := widget.NewLabel("Server stopped")
	status.TextStyle.Bold = true

	ipLabel := widget.NewLabel("IP: -")
	urlLabel := widget.NewLabel("URL: -")

	qrImg := canvas.NewImageFromImage(nil)
	qrImg.FillMode = canvas.ImageFillContain
	qrImg.SetMinSize(fyne.NewSize(220, 220))

	// startBtn variable must be declared first for the closure to access it
	var startBtn *widget.Button 
	
	startBtn = widget.NewButton("Start Server", func() {
		if !state.Running() {

			// START SERVER
			if passwordEntry.Text == "" {
				dialog.ShowInformation("Error", "Enter password", win)
				return
			}

			state.SetPassword(passwordEntry.Text)
			state.SetDevice(deviceEntry.Text)
			startServer()

			ipLabel.SetText("IP: " + primaryIP())
			urlLabel.SetText("URL: " + baseURL())
			qrImg.Image = makeQR(baseURL())
			qrImg.Refresh()

			status.SetText("Server running")
			startBtn.SetText("Stop Server") 

		} else {

			// STOP SERVER
			stopServer()
			ipLabel.SetText("IP: -")
			urlLabel.SetText("URL: -")
			qrImg.Image = nil
			qrImg.Refresh()

			status.SetText("Server stopped")
			startBtn.SetText("Start Server") 
		}
	})

	copyBtn := widget.NewButton("Copy URL", func() {
		myApp.Clipboard().SetContent(baseURL())
	})

	home := container.NewVBox(
		widget.NewLabel("PC Power Link"),
		widget.NewLabel("Device name"),
		deviceEntry,
		widget.NewLabel("Password"),
		passwordEntry,
		startBtn,
		status,
		widget.NewSeparator(),
		container.NewCenter(qrImg),
		urlLabel,
		copyBtn,
		ipLabel,
	)

	about := widget.NewLabel("PC Power Link\nMinimal Version\nv" + Version)

	tabs := container.NewAppTabs(
		container.NewTabItem("Home", container.NewVScroll(home)),
		container.NewTabItem("About", about),
	)

	win.SetContent(tabs)
	win.ShowAndRun()
}
