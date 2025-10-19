package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/d1"
	"github.com/cloudflare/cloudflare-go/v4/kv"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/workers"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

// Global variables for Cloudflare client and account details
// These should be initialized in your main function
var (
	cfClient  *cf.API
	cfAccount *cf.Account
	// workerPath should point to the location of your JavaScript worker file.
	workerPath = "./worker.js"
	// cachePath for tldextract
	cachePath = "/tmp/tld.cache"
)

// ScriptUpdateParams defines the structure for the multipart form data.
type ScriptUpdateParams struct {
	AccountID string
	Metadata  ScriptUpdateParamsMetadataForm
}

// ScriptUpdateParamsMetadataForm holds the metadata for the worker script.
type ScriptUpdateParamsMetadataForm struct {
	MainModule        string              `json:"main_module"`
	Bindings          []map[string]string `json:"bindings"`
	CompatibilityDate string              `json:"compatibility_date"`
	CompatibilityFlags []string            `json:"compatibility_flags"`
	// jsPath is a helper field, not part of the JSON metadata.
	jsPath string
}

// MarshalMultipart creates the multipart/form-data body for the worker upload request.
func (sp ScriptUpdateParams) MarshalMultipart() ([]byte, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 1. Create the metadata part (as JSON)
	metadataJSON, err := json.Marshal(sp.Metadata)
	if err != nil {
		return nil, "", fmt.Errorf("error marshalling metadata: %w", err)
	}

	metadataHeaders := textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="metadata"; filename="metadata.json"`},
		"Content-Type":        []string{"application/json"},
	}
	metadataPart, err := writer.CreatePart(metadataHeaders)
	if err != nil {
		return nil, "", fmt.Errorf("error creating metadata part: %w", err)
	}
	_, err = metadataPart.Write(metadataJSON)
	if err != nil {
		return nil, "", fmt.Errorf("error writing metadata content: %w", err)
	}

	// 2. Create the script file part
	fileHeaders := textproto.MIMEHeader{
		"Content-Disposition": []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, sp.Metadata.MainModule, sp.Metadata.MainModule)},
		"Content-Type":        []string{"application/javascript+module"},
	}
	filePart, err := writer.CreatePart(fileHeaders)
	if err != nil {
		return nil, "", fmt.Errorf("error creating file part: %w", err)
	}

	file, err := os.Open(sp.Metadata.jsPath)
	if err != nil {
		return nil, "", fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(filePart, file)
	if err != nil {
		return nil, "", fmt.Errorf("error copying file content: %w", err)
	}

	// Close the writer to finalize the multipart body
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("error closing multipart writer: %w", err)
	}

	return body.Bytes(), writer.FormDataContentType(), nil
}

// createWorker creates or updates a Cloudflare Worker with the necessary bindings.
func createWorker(
	ctx context.Context,
	name string,
	adminKey string,
	proxyIP string,
	rootProxyURL string, // Optional
	kvNamespace *kv.Namespace,
	d1Database *d1.Database,
) (*workers.ScriptUpdateResponse, error) {

	// Define the bindings required by the new JavaScript worker script
	bindings := []map[string]string{
		// D1 Database Binding
		{
			"name":        "DB",
			"type":        "d1_database",
			"database_id": d1Database.UUID,
		},
		// KV Namespace Binding
		{
			"name":         "USER_KV",
			"type":         "kv_namespace",
			"namespace_id": kvNamespace.ID,
		},
		// Secret Bindings
		{
			"name": "ADMIN_KEY",
			"type": "secret_text",
			"text": adminKey,
		},
		{
			"name": "PROXYIP",
			"type": "secret_text",
			"text": proxyIP,
		},
	}

	if rootProxyURL != "" {
		bindings = append(bindings, map[string]string{
			"name": "ROOT_PROXY_URL",
			"type": "secret_text",
			"text": rootProxyURL,
		})
	}

	param := ScriptUpdateParams{
		AccountID: cfAccount.ID,
		Metadata: ScriptUpdateParamsMetadataForm{
			MainModule:        "worker.js",
			Bindings:          bindings,
			jsPath:            workerPath,
			CompatibilityDate: time.Now().Format("2006-01-02"),
			CompatibilityFlags: []string{
				"nodejs_compat",
			},
		},
	}

	data, contentType, err := param.MarshalMultipart()
	if err != nil {
		return nil, fmt.Errorf("error marshalling multipart data: %w", err)
	}
	r := bytes.NewReader(data)

	// Use the SDK to upload the script
	result, err := cfClient.Workers.Scripts.Update(
		ctx,
		name,
		workers.ScriptUpdateParams{AccountID: cf.String(cfAccount.ID)},
		option.WithRequestBody(contentType, r),
	)
	if err != nil {
		return nil, fmt.Errorf("error updating worker script: %w", err)
	}

	return result, nil
}

// createKVNamespace creates a new KV namespace.
func createKVNamespace(ctx context.Context, nsName string) (*kv.Namespace, error) {
	fmt.Printf("Creating KV Namespace '%s'...\n", nsName)
	res, err := cfClient.KV.Namespaces.New(ctx, kv.NamespaceNewParams{AccountID: cf.String(cfAccount.ID), Title: cf.String(nsName)})
	if err != nil {
		return nil, fmt.Errorf("error creating KV namespace: %w", err)
	}
	fmt.Println("KV Namespace created successfully.")
	return res, nil
}

// createD1Database creates a new D1 database.
func createD1Database(ctx context.Context, dbName string) (*d1.Database, error) {
	fmt.Printf("Creating D1 Database '%s'...\n", dbName)
	res, err := cfClient.D1.New(ctx, d1.DatabaseNewParams{AccountID: cf.String(cfAccount.ID), Name: cf.String(dbName)})
	if err != nil {
		return nil, fmt.Errorf("error creating D1 database: %w", err)
	}
	fmt.Println("D1 Database created successfully.")
	return res, nil
}

// enableWorkerSubdomain enables the workers.dev subdomain for the account.
func enableWorkerSubdomain(ctx context.Context) error {
	fmt.Println("Enabling workers.dev subdomain...")
	_, err := cfClient.Workers.Subdomains.New(
		ctx,
		workers.SubdomainNewParams{
			AccountID: cf.String(cfAccount.ID),
			Body:      cf.String("enabled"),
		})
	// We can ignore "subdomain already exists" error
	if err != nil && !strings.Contains(err.Error(), "subdomain already exists") {
		return fmt.Errorf("error enabling worker subdomain: %w", err)
	}
	fmt.Println("Subdomain is enabled.")
	return nil
}

// addWorkerCustomDomain attaches a custom domain to a worker.
func addWorkerCustomDomain(ctx context.Context, scriptName string, customDomain string) (string, error) {
	fmt.Printf("Attaching custom domain '%s' to worker '%s'...\n", customDomain, scriptName)
	extractor, err := tldextract.New(cachePath, false)
	if err != nil {
		return "", fmt.Errorf("error initializing tldextract: %w", err)
	}

	result := extractor.Extract(customDomain)
	domain := fmt.Sprintf("%s.%s", result.Root, result.Tld)

	zones, err := cfClient.Zones.List(ctx, zones.ZoneListParams{
		Account: cf.Account{ID: cfAccount.ID},
		Name:    cf.String(domain),
	})
	if err != nil {
		return "", fmt.Errorf("error listing zones: %w", err)
	}
	if len(zones.Result) == 0 {
		return "", fmt.Errorf("no zone found for domain: %s", domain)
	}
	zone := zones.Result[0]

	res, err := cfClient.Workers.Domains.Update(ctx, workers.DomainUpdateParams{
		AccountID:   cf.String(cfAccount.ID),
		Environment: cf.String("production"),
		Hostname:    cf.String(customDomain),
		Service:     cf.String(scriptName),
		ZoneID:      cf.String(zone.ID),
	})
	if err != nil {
		return "", fmt.Errorf("error updating worker domain: %w", err)
	}
	fmt.Println("Custom domain attached successfully.")
	return res.Hostname, nil
}

// isWorkerAvailable checks if a worker name is already taken.
func isWorkerAvailable(ctx context.Context, name string) bool {
	_, err := cfClient.Workers.Scripts.Get(ctx, name, workers.ScriptGetParams{AccountID: cf.String(cfAccount.ID)})
	return err != nil // If there is an error (like Not Found), the name is available.
}

// deployWorker orchestrates the entire deployment process.
func deployWorker(
	ctx context.Context,
	name string,
	adminKey string,
	proxyIP string,
	rootProxyURL string,
	customDomain string,
) (panelURL string, err error) {
	// 1. Create a KV Namespace for the worker
	kvNamespaceName := fmt.Sprintf("USER_KV_%s", name)
	kvNamespace, err := createKVNamespace(ctx, kvNamespaceName)
	if err != nil {
		return "", err
	}

	// 2. Create a D1 Database for the worker
	d1DatabaseName := fmt.Sprintf("DB_%s", name)
	d1Database, err := createD1Database(ctx, d1DatabaseName)
	if err != nil {
		return "", err
	}

	// 3. Create and upload the worker script with bindings
	fmt.Printf("Creating worker '%s'...\n", name)
	_, err = createWorker(ctx, name, adminKey, proxyIP, rootProxyURL, kvNamespace, d1Database)
	if err != nil {
		return "", fmt.Errorf("failed to deploy worker: %w", err)
	}
	fmt.Println("Worker created successfully!")

	// 4. Enable the workers.dev subdomain for the account
	err = enableWorkerSubdomain(ctx)
	if err != nil {
		return "", err
	}

	// 5. Handle domain assignment
	if customDomain != "" {
		_, err := addWorkerCustomDomain(ctx, name, customDomain)
		if err != nil {
			return "", fmt.Errorf("failed to add custom domain: %w", err)
		}
		// The admin panel path is hardcoded in the JS script as /admin or from ADMIN_PATH env var.
		return "https://" + customDomain + "/admin", nil
	}

	// If no custom domain, use the workers.dev subdomain
	resp, err := cfClient.Workers.Subdomains.Get(ctx, workers.SubdomainGetParams{AccountID: cf.String(cfAccount.ID)})
	if err != nil {
		return "", fmt.Errorf("error getting worker subdomain: %w", err)
	}

	return "https://" + name + "." + resp.Subdomain + "/admin", nil
}

// promptUser is a helper to get input from the user.
func promptUser(promptText string) string {
	fmt.Print(promptText)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

// Main function to run the application
func main() {
	// --- Configuration ---
	// It's recommended to load these from environment variables or a config file.
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	if apiToken == "" || accountID == "" {
		log.Fatal("Please set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID environment variables.")
	}

	// Check if the worker script file exists
	if _, err := os.Stat(workerPath); os.IsNotExist(err) {
		log.Fatalf("Worker script not found at '%s'. Please make sure the file exists.", workerPath)
	}

	var err error
	cfClient, err = cf.New(apiToken, cf.WithAPIToken(apiToken))
	if err != nil {
		log.Fatalf("Error creating Cloudflare client: %v", err)
	}

	cfAccount = &cf.Account{ID: accountID}
	ctx := context.Background()

	// --- Gather Information from User ---
	fmt.Println("--- Cloudflare VLESS Worker Deployment ---")
	workerName := promptUser("Enter a name for your new worker: ")
	adminKey := promptUser("Enter the admin panel password (ADMIN_KEY): ")
	proxyIP := promptUser("Enter a clean IP/domain for configs (PROXYIP): ")
	rootProxyURL := promptUser("Enter a URL to proxy at root '/' (optional, press Enter to skip): ")
	customDomain := promptUser("Enter a custom domain (optional, press Enter to use workers.dev): ")

	if workerName == "" || adminKey == "" || proxyIP == "" {
		log.Fatal("Worker name, admin key, and proxy IP are required.")
	}

	// Check if worker name is available
	if !isWorkerAvailable(ctx, workerName) {
		log.Fatalf("Worker name '%s' is already taken. Please choose another name.", workerName)
	}

	// --- Deploy ---
	fmt.Println("\nStarting deployment process...")
	panelURL, err := deployWorker(ctx, workerName, adminKey, proxyIP, rootProxyURL, customDomain)
	if err != nil {
		log.Fatalf("\nDeployment failed: %v", err)
	}

	fmt.Println("\n--- ✅ Deployment Successful! ---")
	fmt.Printf("Admin Panel URL: %s\n", panelURL)
	fmt.Println("Please wait a minute for all changes to propagate.")
	fmt.Println("You will need to run the DB initialization command using wrangler on your local machine.")
	fmt.Printf("Example command:\nwrangler d1 execute DB_%s --command=\"CREATE TABLE IF NOT EXISTS users (uuid TEXT PRIMARY KEY, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, expiration_date TEXT NOT NULL, expiration_time TEXT NOT NULL, notes TEXT, data_limit INTEGER DEFAULT 0, data_usage INTEGER DEFAULT 0, ip_limit INTEGER DEFAULT 2);\"\n", workerName)
}
