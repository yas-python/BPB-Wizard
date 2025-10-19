package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =====================
// Terminal color codes
// =====================
const (
	RED    = "1"
	GREEN  = "2"
	ORANGE = "208"
	BLUE   = "39"
)

// =====================
// Global variables
// =====================
var (
	title      = fmtStr("●", BLUE, true)
	ask        = fmtStr("-", "", true)
	info       = fmtStr("+", "", true)
	warning    = fmtStr("Warning", RED, true)
	srcPath    string
	workerPath string
	cachePath  string
	isAndroid  bool
	VERSION    = "1.0.0" // default; override at build with -ldflags "-X main.VERSION=..."
)

// checkAndroid detects Termux/Android and sets SSL_CERT_FILE accordingly.
func checkAndroid() {
	path := os.Getenv("PATH")
	if runtime.GOOS == "android" || strings.Contains(path, "com.termux") {
		prefix := os.Getenv("PREFIX")
		certPath := filepath.Join(prefix, "etc/tls/cert.pem")
		if err := os.Setenv("SSL_CERT_FILE", certPath); err != nil {
			failMessage("Failed to set Termux certificate file.")
			log.Fatalln(err)
		}
		isAndroid = true
	}
}

// setDNS forces the default HTTP transport to use a custom DNS resolver (UDP 8.8.8.8:53).
// This is helpful when system resolver is unreliable (or when behind VPNs).
func setDNS() {
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		d := net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					conn, err := net.Dial("udp", "8.8.8.8:53")
					if err != nil {
						failMessage("Failed to dial DNS. Please disconnect your VPN and try again.")
						log.Fatal(err)
					}
					return conn, nil
				},
			},
		}
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			failMessage("DNS resolution failed. Please disconnect your VPN and try again.")
			log.Fatal(err)
		}
		return conn, nil
	}
}

// initPaths creates a temporary workspace and defines worker/cache paths.
func initPaths() {
	var err error
	srcPath, err = os.MkdirTemp("", ".bpb-wizard")
	if err != nil {
		failMessage("Failed to create temporary directory.")
		log.Fatalln(err)
	}

	workerPath = filepath.Join(srcPath, "worker.js")
	cachePath = filepath.Join(srcPath, "tld.cache")
}

// renderHeader prints an ASCII header with the current version.
func renderHeader() {
	fmt.Printf(`
■■■■■■■  ■■■■■■■  ■■■■■■■ 
■■   ■■  ■■   ■■  ■■   ■■
■■■■■■■  ■■■■■■■  ■■■■■■■ 
■■   ■■  ■■       ■■   ■■
■■■■■■■  ■■       ■■■■■■■  %s %s
`,
		fmtStr("Wizard", GREEN, true),
		fmtStr(VERSION, GREEN, false),
	)
}

// fmtStr renders a string with lipgloss style (color + bold option).
func fmtStr(str string, color string, isBold bool) string {
	style := lipgloss.NewStyle().Bold(isBold)
	if color != "" {
		style = style.Foreground(lipgloss.Color(color))
	}
	return style.Render(str)
}

// failMessage prints an error prefix and the provided message.
func failMessage(msg string) {
	fmt.Println(fmtStr("[ERROR]", RED, true), msg)
}
