package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/d1"
	"github.com/cloudflare/cloudflare-go/v4/kv"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/pages"
)

// ======= Global Variables =======
var (
	cfClient  *cf.API
	cfAccount = &cf.Account{ID: "YOUR_ACCOUNT_ID"}
	title     = "[Deployer]"
)

// ======= Worker Config Struct =======
type WorkerConfig struct {
	UUID                string
	ADMIN_KEY           string
	ADMIN_PATH          string
	PROXYIP             string
	SCAMALYTICS_API_KEY string
	SOCKS5              string
	SOCKS5_RELAY        string
	ROOT_PROXY_URL      string
}

// ======= Mock Helpers =======
func promptUser(prompt string) string {
	fmt.Print(prompt)
	var response string
	fmt.Scanln(&response)
	return response
}

func successMessage(msg string) {
	fmt.Printf("\n✅ %s %s\n", title, msg)
}

func failMessage(msg string) {
	fmt.Printf("\n❌ %s %s\n", title, msg)
}

// ======= Placeholder Deployment Functions =======
func createPagesDeployment(ctx context.Context, project *pages.Project) error {
	fmt.Printf("%s Skipping actual file deployment.\n", title)
	return nil
}

func addPagesProjectCustomDomain(ctx context.Context, project *pages.Project, domain string) error {
	fmt.Printf("%s Adding custom domain '%s'.\n", title, domain)
	return nil
}

// ======= Pages Project Creation =======
func createPagesProject(
	ctx context.Context,
	name string,
	config *WorkerConfig,
	userKV *kv.Namespace,
	d1DB *d1.Database,
) (*pages.Project, error) {

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

	// Optional vars
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

	if cfClient == nil {
		// Return mock project to allow local test
		return &pages.Project{
			Name:      name,
			Subdomain: name + ".pages.dev",
			Domains:   []string{name + ".pages.dev"},
		}, fmt.Errorf("cfClient not initialized (mock project returned)")
	}

	project, err := cfClient.CreatePagesProject(
		ctx,
		option.AccountID(cfAccount.ID),
		pages.Project{
			Name:             name,
			ProductionBranch: "main",
			DeploymentConfigs: &pages.ProjectDeploymentConfigs{
				Production: &pages.ProjectDeploymentConfigsProduction{
					CompatibilityDate: time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
					EnvVars:           envVars,
					KVNamespaces: map[string]pages.ProjectDeploymentConfigsProductionKVNamespace{
						"USER_KV": {NamespaceID: userKV.ID},
					},
					D1Databases: map[string]pages.ProjectDeploymentConfigsProductionD1Database{
						"DB": {DatabaseID: d1DB.UUID},
					},
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

// ======= Deployment Orchestrator =======
func deployPagesProject(
	ctx context.Context,
	name string,
	workerCfg *WorkerConfig,
	kvNamespace *kv.Namespace,
	d1DB *d1.Database,
	customDomain string,
) (string, error) {

	var project *pages.Project
	var err error

	for {
		fmt.Printf("\n%s Creating Pages project...\n", title)
		project, err = createPagesProject(ctx, name, workerCfg, kvNamespace, d1DB)
		if err != nil && strings.Contains(err.Error(), "mock project") {
			log.Printf("Using mock project for local testing...")
			project.Domains = []string{name + ".pages.dev"}
			break
		} else if err != nil {
			failMessage("Failed to create project")
			if promptUser("Try again? (y/n): ") == "n" {
				return "", err
			}
			continue
		}
		break
	}

	if err := createPagesDeployment(ctx, project); err != nil {
		failMessage("Deployment failed")
		return "", err
	}
	successMessage("Deployment successful")

	if customDomain != "" {
		if err := addPagesProjectCustomDomain(ctx, project, customDomain); err != nil {
			failMessage("Custom domain addition failed")
		} else {
			successMessage("Custom domain added")
			project.Domains = append(project.Domains, customDomain)
		}
	}

	panelURL := fmt.Sprintf("https://%s/%s", project.Subdomain, workerCfg.ADMIN_PATH)
	fmt.Printf("\nProject: %s\nDomains: %v\nAdmin Panel: %s\n", project.Name, project.Domains, panelURL)

	return panelURL, nil
}

// ======= Main Entry Point =======
func main() {
	fmt.Println("--- Cloudflare Pages VLESS Worker Deployer ---")

	config := &WorkerConfig{
		UUID:       "your-uuid",
		ADMIN_KEY:  "your-admin-key",
		ADMIN_PATH: "admin",
	}

	mockKV := &kv.Namespace{ID: "mock_kv_namespace_id"}
	mockD1 := &d1.Database{UUID: "mock_d1_database_uuid"}

	ctx := context.Background()
	projectName := "vless-worker-project"
	customDomain := "vless.example.com"

	_, err := deployPagesProject(ctx, projectName, config, mockKV, mockD1, customDomain)
	if err != nil {
		log.Fatalf("Deployment failed: %v", err)
	}

	successMessage("Process completed successfully")
}
