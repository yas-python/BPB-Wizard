package main

import (
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

	cf "github.com/cloudflare/cloudflare-go"
	cfd1 "github.com/cloudflare/cloudflare-go/d1"
	"github.com/cloudflare/cloudflare-go/kv"
	"github.com/cloudflare/cloudflare-go/option"
	"github.com/cloudflare/cloudflare-go/pages"
)

// --- Mock/Placeholder Variables and Functions ---
// These are placeholders to make the code runnable.
// You should replace them with your actual implementations.

var (
	// cfClient would be your initialized Cloudflare API client
	cfClient *cf.API
	// cfAccount would be your fetched Cloudflare account details
	cfAccount = &cf.Account{ID: "YOUR_ACCOUNT_ID"}
	// title is a helper for printing formatted messages
	title = "[Deployer]"
)

// Mock implementation of a user prompt
func promptUser(prompt string) string {
	fmt.Print(prompt)
	var response string
	fmt.Scanln(&response)
	return response
}

// Mock implementation for success messages
func successMessage(msg string) {
	fmt.Printf("\n✅ %s %s\n", title, msg)
}

// Mock implementation for failure messages
func failMessage(msg string) {
	fmt.Printf("\n❌ %s %s\n", title, msg)
}

// Placeholder for your actual deployment logic
func createPagesDeployment(ctx context.Context, project *pages.Project) error {
	fmt.Printf("%s Skipping actual file deployment in this example.\n", title)
	// In your real code, you would upload the worker file here.
	return nil
}

// Placeholder for your custom domain logic
func addPagesProjectCustomDomain(ctx context.Context, project *pages.Project, domain string) error {
	fmt.Printf("%s Adding custom domain '%s' to project '%s'.\n", title, domain, project.Name)
	// Your real code to add the domain would go here.
	return nil
}

// --- End of Mock/Placeholder Section ---

// projectDeploymentNewParams is a placeholder for your deployment parameters struct.
// The user's code referenced this, so it's included for completeness.
type projectDeploymentNewParams struct {
	// Define fields based on what you need to marshal
	Manifest string
}

// MarshalMultipart is a placeholder method.
// The user's code referenced this, so it's included for completeness.
func (p projectDeploymentNewParams) MarshalMultipart(w *multipart.Writer) error {
	// Your logic to write parts to the multipart writer
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="manifest.json"`)
	h.Set("Content-Type", "application/json")
	pw, err := w.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = pw.Write([]byte(p.Manifest))
	return err
}

// WorkerConfig struct to hold the settings required by the VLESS worker
type WorkerConfig struct {
	UUID                string
	ADMIN_KEY           string
	ADMIN_PATH          string
	PROXYIP             string
	SCAMALYTICS_API_KEY string
	SOCKS5              string
	SOCKS5_RELAY        string // Should be "true" or "false"
	ROOT_PROXY_URL      string
}

// createPagesProject creates a new Cloudflare Pages project with the specified configuration.
// This function is one of the main pieces of code provided by the user.
func createPagesProject(
	ctx context.Context,
	name string,
	config *WorkerConfig, // Struct containing worker settings
	userKV *kv.Namespace, // Renamed param for clarity
	d1DB *cfd1.Database,  // New parameter for the D1 binding
) (
	*pages.Project,
	error,
) {
	// Build the environment variable map for the VLESS worker
	envVars := map[string]pages.ProjectDeploymentConfigsProductionEnvVarsUnionParam{
		"UUID": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.UUID),
		},
		"ADMIN_KEY": pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.ADMIN_KEY),
		},
	}

	// Add optional variables only if they are set
	if config.ADMIN_PATH != "" {
		envVars["ADMIN_PATH"] = pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.ADMIN_PATH),
		}
	}
	if config.PROXYIP != "" {
		envVars["PROXYIP"] = pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.PROXYIP),
		}
	}
	if config.SCAMALYTICS_API_KEY != "" {
		envVars["SCAMALYTICS_API_KEY"] = pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.SCAMALYTICS_API_KEY),
		}
	}
	if config.SOCKS5 != "" {
		envVars["SOCKS5"] = pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.SOCKS5),
		}
	}
	if config.SOCKS5_RELAY != "" {
		envVars["SOCKS5_RELAY"] = pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.SOCKS5_RELAY),
		}
	}
	if config.ROOT_PROXY_URL != "" {
		envVars["ROOT_PROXY_URL"] = pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarParam{
			Type:  cf.F(pages.ProjectDeploymentConfigsProductionEnvVarsPagesPlainTextEnvVarTypePlainText),
			Value: cf.F(config.ROOT_PROXY_URL),
		}
	}

	// This is a placeholder call. In a real scenario, cfClient would be initialized.
	// To run this, you would need to initialize cfClient first:
	// api, err := cf.New("YOUR_API_TOKEN", "YOUR_EMAIL", option.WithAPIToken("YOUR_API_TOKEN"))
	if cfClient == nil {
		return &pages.Project{Name: name, Subdomain: name + ".pages.dev"}, fmt.Errorf("cfClient is not initialized (this is expected in the example)")
	}

	project, err := cfClient.Pages.Projects.New(
		ctx,
		pages.ProjectNewParams{
			AccountID: cf.F(cfAccount.ID),
			Project: pages.ProjectParam{
				Name:             cf.F(name),
				ProductionBranch: cf.F("main"),
				DeploymentConfigs: cf.F(pages.ProjectDeploymentConfigsParam{
					Production: cf.F(pages.ProjectDeploymentConfigsProductionParam{
						CompatibilityDate:  cf.F(time.Now().AddDate(0, 0, -1).Format("2006-01-02")),
						CompatibilityFlags: cf.F([]string{"nodejs_compat"}),
						// Set the correct KV Namespace binding (as expected by the JS worker)
						KVNamespaces: cf.F(map[string]pages.ProjectDeploymentConfigsProductionKVNamespaceParam{
							"USER_KV": { // <-- Name must be USER_KV
								NamespaceID: cf.F(userKV.ID),
							},
						}),
						// Set the D1 Database binding (as expected by the JS worker)
						D1Databases: cf.F(map[string]pages.ProjectDeploymentConfigsProductionD1DatabaseParam{
							"DB": { // <-- Name must be DB
								DatabaseID: cf.F(d1DB.UUID),
							},
						}),
						EnvVars: cf.F(envVars), // <-- Updated environment variables
					}),
				}),
			},
		})

	if err != nil {
		return nil, fmt.Errorf("error creating pages project: %w", err)
	}

	return project, nil
}

// deployPagesProject orchestrates the creation and deployment of a Pages project.
// This function is the second main piece of code provided by the user.
func deployPagesProject(
	ctx context.Context,
	name string,
	workerCfg *WorkerConfig, // Pass the new struct
	kvNamespace *kv.Namespace,
	d1DB *cfd1.Database, // Pass the D1 database object
	customDomain string,
) (
	panelURL string,
	er error,
) {
	var project *pages.Project
	var err error

	for {
		fmt.Printf("\n%s Creating Pages project...\n", title)

		// Call the modified function to create the project
		project, err = createPagesProject(ctx, name, workerCfg, kvNamespace, d1DB)

		// The placeholder createPagesProject returns an error, so we handle it gracefully here
		// In a real run, this would only catch genuine API errors
		if err != nil && strings.Contains(err.Error(), "cfClient is not initialized") {
			log.Printf("Continuing with mock project due to uninitialized client...")
			project = &pages.Project{
				Name:      name,
				Subdomain: name + ".pages.dev",
				Domains:   []string{name + ".pages.dev"},
			}
		} else if err != nil {
			failMessage("Failed to create project.")
			log.Printf("%v\n\n", err)
			if response := promptUser("Would you like to try again? (y/n): "); strings.ToLower(response) == "n" {
				return "", nil
			}
			continue
		}

		successMessage("Page created successfully!")
		break
	}

	// After creating the project, you would deploy the worker code
	if err := createPagesDeployment(ctx, project); err != nil {
		failMessage("Deployment failed.")
		log.Printf("%v\n", err)
		return "", err
	}
	successMessage("Deployment successful!")

	// Finally, add the custom domain if provided
	if customDomain != "" {
		if err := addPagesProjectCustomDomain(ctx, project, customDomain); err != nil {
			failMessage("Failed to add custom domain.")
			log.Printf("%v\n", err)
			// Don't fail the whole process if domain addition fails
		} else {
			successMessage("Custom domain added successfully!")
			project.Domains = append(project.Domains, customDomain)
		}
	}

	// Construct the panel URL
	finalURL := "https://" + project.Subdomain
	panelURL = fmt.Sprintf("%s/%s", finalURL, workerCfg.ADMIN_PATH)

	fmt.Printf("\n--- Project Details ---\n")
	fmt.Printf("Project Name: %s\n", project.Name)
	fmt.Printf("Domains: %s\n", strings.Join(project.Domains, ", "))
	fmt.Printf("Admin Panel: %s\n", panelURL)
	fmt.Printf("-----------------------\n")

	return panelURL, nil
}

// Main function to demonstrate the usage
func main() {
	fmt.Println("--- Cloudflare Pages VLESS Worker Deployer ---")

	// 1. Define the worker configuration
	config := &WorkerConfig{
		UUID:       "your-unique-uuid-here",
		ADMIN_KEY:  "your-secret-admin-key",
		ADMIN_PATH: "admin", // The path for the admin panel
		// Other fields can be left empty if not needed
	}

	// 2. Define mock KV and D1 resources (in a real app, you'd create/fetch these via API)
	mockKV := &kv.Namespace{
		ID:    "mock_kv_namespace_id_12345",
		Title: "My-User-KV",
	}
	mockD1 := &cfd1.Database{
		UUID: "mock_d1_database_uuid_67890",
		Name: "My-DB",
	}

	// 3. Set project details
	projectName := "vless-worker-project"
	domain := "vless.yourdomain.com" // Set to "" to skip adding a custom domain

	// 4. Call the main deployment function
	ctx := context.Background()
	_, err := deployPagesProject(ctx, projectName, config, mockKV, mockD1, domain)
	if err != nil {
		log.Fatalf("Deployment process failed: %v", err)
	}

	successMessage("Process completed.")
}
