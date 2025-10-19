package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/accounts"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"golang.org/x/oauth2"
)

//go:embed static/index.html
var indexHTML []byte

// ======= Global Variables =======
var (
	cfClient      *cf.Client
	cfAccount     *accounts.Account
	state         string
	codeVerifier  string
	obtainedToken = make(chan *oauth2.Token)
	title         = "[Deployer]"
	ORANGE        = "208"
)

// OAuth2 Config
var config = &oauth2.Config{
	ClientID:     "54d11594-84e4-41aa-b438-e81b8fa78ee7",
	ClientSecret: "",
	RedirectURL:  "http://localhost:8976/oauth/callback",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://dash.cloudflare.com/oauth2/auth",
		TokenURL: "https://dash.cloudflare.com/oauth2/token",
	},
	Scopes: []string{
		"account:read", "user:read", "workers:write", "workers_kv:write",
		"workers_routes:write", "workers_scripts:write", "workers_tail:read",
		"d1:write", "pages:write", "pages:read", "zone:read", "ssl_certs:write",
		"ai:write", "queues:write", "pipelines:write", "secrets_store:write",
	},
}

// ======= Helper Functions =======
func fmtStr(str, color string, bold bool) string {
	return str // simple version; could integrate termcolor or lipgloss
}

func failMessage(msg string) {
	fmt.Printf("\n❌ %s %s\n", title, msg)
}

func successMessage(msg string) {
	fmt.Printf("\n✅ %s %s\n", title, msg)
}

func openURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// ======= OAuth2 Helper Functions =======
func generateState() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	buff := make([]byte, 16)
	for i := range buff {
		buff[i] = charset[rand.Intn(len(charset))]
	}
	return base64.URLEncoding.EncodeToString(buff)
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func generateAuthURL() string {
	state = generateState()
	codeVerifier = generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	return config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
}

// ======= OAuth Login and Callback =======
func login() {
	url := generateAuthURL()
	fmt.Printf("\n%s Login %s...\n", title, fmtStr("Cloudflare", ORANGE, true))

	if err := openURL(url); err != nil {
		failMessage("Failed to open browser for login")
		log.Fatalln(err)
	}
}

func callback(w http.ResponseWriter, r *http.Request) {
	param := r.URL.Query().Get("state")
	if param != state {
		failMessage("Invalid OAuth state")
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		failMessage("No code returned")
		http.Error(w, "No code", http.StatusBadRequest)
		return
	}

	token, err := config.Exchange(context.Background(), code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		failMessage("Failed to exchange token")
		log.Fatalln(err)
	}

	obtainedToken <- token
	successMessage("Cloudflare logged in successfully")

	w.Header().Set("Content-Type", "text/html")
	w.Write(indexHTML)
}

// ======= Cloudflare Account Fetch =======
func getAccount(ctx context.Context) (*accounts.Account, error) {
	if cfClient == nil {
		return nil, fmt.Errorf("cfClient not initialized")
	}
	res, err := cfClient.Accounts.List(ctx, accounts.AccountListParams{})
	if err != nil {
		return nil, fmt.Errorf("error listing accounts: %v", err)
	}
	return &res.Result[0], nil
}

// ======= Main Entry =======
func main() {
	rand.Seed(time.Now().UnixNano())
	login()

	http.HandleFunc("/oauth/callback", callback)
	server := &http.Server{Addr: ":8976"}

	fmt.Println("--- OAuth Server Listening on http://localhost:8976 ---")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		failMessage("Server error")
		log.Fatalln(err)
	}
}
