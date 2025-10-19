module BPB-Wizard

go 1.24.1

// --- Direct Dependencies ---
// All the packages your project directly uses are listed here.
// I have corrected the cloudflare-go path and updated it to a recent, stable version.
require (
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/cloudflare/cloudflare-go v0.99.0
	github.com/google/uuid v1.6.0
	github.com/joeguo/tldextract v0.0.0-20220507100122-d83daa6adef8
	golang.org/x/oauth2 v0.30.0
)

// --- Indirect Dependencies ---
// These are dependencies of your dependencies.
// Go will manage this section automatically when you run 'go mod tidy'.
require (
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.3.2 // indirect
	github.com/charmbracelet/x/ansi v0.10.1 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/time v0.5.0 // indirect
)
