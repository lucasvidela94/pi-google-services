package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sombi/pi-google-services/internal/auth"
	"github.com/sombi/pi-google-services/internal/calendar"
	"github.com/sombi/pi-google-services/internal/config"
	"github.com/sombi/pi-google-services/internal/gmail"
	"github.com/sombi/pi-google-services/internal/mcp"
	"github.com/sombi/pi-google-services/internal/services"
)

const version = "0.1.0"

//go:embed credentials.json
var embeddedCreds []byte

func main() {
	log.SetFlags(0)
	log.SetPrefix("pi-google: ")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "login":
		cmdLogin()
	case "logout":
		cmdLogout()
	case "serve":
		cmdServe()
	case "status":
		cmdStatus()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`pi-google-services v%s — Google services MCP server for Pi

Usage:
  pi-google-services login          Authenticate with Google (browser)
  pi-google-services logout         Remove stored credentials
  pi-google-services serve          Start MCP server (stdio)
  pi-google-services status         Show auth status
  pi-google-services help           Show this help

The OAuth client ID is embedded. Run 'login' once, then 'serve'.

Add to Pi MCP config (~/.pi/agent/mcp.json):
  { "mcpServers": {
      "google-services": {
        "command": "pi-google-services",
        "args": ["serve"] }
    } }
`, version)
}

// getCredentialsJSON returns embedded or file-based credentials.
func getCredentialsJSON() ([]byte, error) {
	if len(embeddedCreds) > 0 {
		return embeddedCreds, nil
	}
	if path := os.Getenv("GOOGLE_OAUTH_CREDENTIALS"); path != "" {
		return os.ReadFile(path)
	}
	if dir, err := config.Dir(); err == nil {
		if data, err := os.ReadFile(filepath.Join(dir, "credentials.json")); err == nil {
			return data, nil
		}
	}
	if data, err := os.ReadFile("credentials.json"); err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("no credentials found (embedded, env, or file)")
}

// registeredServices returns all available services with their scopes.
func registeredServices() []services.Service {
	return []services.Service{
		services.NewCalendar(nil), // placeholder; api set during serve
		services.NewGmail(nil),
	}
}

// allScopes aggregates scopes from all registered services.
func allScopes() []string {
	seen := map[string]bool{}
	var scopes []string
	for _, svc := range registeredServices() {
		for _, s := range svc.Scopes() {
			if !seen[s] {
				seen[s] = true
				scopes = append(scopes, s)
			}
		}
	}
	return scopes
}

// All tools aggregated from all services.
func allTools() []mcp.ToolDefinition {
	var tools []mcp.ToolDefinition
	for _, svc := range registeredServices() {
		tools = append(tools, svc.Tools()...)
	}
	return tools
}

func cmdLogin() {
	credsData, err := getCredentialsJSON()
	if err != nil {
		fmt.Println("❌ No se encontraron credenciales.")
		fmt.Println("   Seteá GOOGLE_OAUTH_CREDENTIALS o copiá credentials.json")
		os.Exit(1)
	}

	creds, err := config.LoadCredentialsFromBytes(credsData)
	if err != nil {
		log.Fatalf("Credenciales inválidas: %v", err)
	}

	// Build auth with all service scopes
	scopes := allScopes()
	cfg := &config.Config{
		ClientID:     creds.Installed.ClientID,
		ClientSecret: creds.Installed.ClientSecret,
		Scopes:       scopes,
	}

	a := auth.NewFromConfig(cfg)
	ctx := context.Background()

	fmt.Println("\n🔐 Abriendo navegador para autorizar con Google...")
	fmt.Printf("   Scopes solicitados: %d servicios\n", len(registeredServices()))
	token, err := a.Login(ctx)
	if err != nil {
		log.Fatalf("Login falló: %v", err)
	}

	if err := cfg.Save(); err != nil {
		log.Printf("Warning: no se pudo guardar config: %v", err)
	}

	showLen := len(token.AccessToken)
	if showLen > 10 {
		showLen = 10
	}
	fmt.Printf("\n✅ Login exitoso!\n")
	fmt.Printf("   Access token: %s…\n", token.AccessToken[:showLen])
	fmt.Printf("   Refresh token: %v\n", token.RefreshToken != "")
	fmt.Printf("\nAhora corré:  %s serve\n", os.Args[0])
}

func cmdLogout() {
	dir, err := config.Dir()
	if err != nil {
		log.Fatalf("Config dir: %v", err)
	}
	tokenPath := filepath.Join(dir, config.TokenFile)
	if err := os.Remove(tokenPath); os.IsNotExist(err) {
		fmt.Println("No hay credenciales.")
		return
	} else if err != nil {
		log.Fatalf("Error al remover: %v", err)
	}
	fmt.Println("✅ Credenciales eliminadas.")
}

func cmdServe() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config: %v", err)
	}

	token, err := auth.LoadToken()
	if err != nil {
		log.Fatalf("Token: %v", err)
	}
	if token == nil {
		log.Fatal("❌ No autenticado. Corré 'pi-google-services login' primero.")
	}

	// Ensure client ID is configured
	if cfg.ClientID == "" {
		credsData, err := getCredentialsJSON()
		if err == nil {
			if creds, err := config.LoadCredentialsFromBytes(credsData); err == nil {
				a := auth.NewFromCredentials(creds)
				cfg.ClientID = a.OAuthConfig.ClientID
				cfg.ClientSecret = a.OAuthConfig.ClientSecret
				cfg.Scopes = allScopes()
			}
		}
	}
	if cfg.ClientID == "" {
		log.Fatal("❌ No client ID. Corré 'pi-google-services login' primero.")
	}

	a := auth.NewFromConfig(cfg)
	ctx := context.Background()
	ts := a.TokenSource(ctx, token)

	// Create services
	calSvc, err := calendar.New(ctx, ts)
	if err != nil {
		log.Fatalf("Calendar: %v", err)
	}

	gmailSvc, err := gmail.New(ctx, ts)
	if err != nil {
		log.Fatalf("Gmail: %v", err)
	}

	// Build MCP server with registered services
	server := mcp.New()
	registerServiceTools(server, services.NewCalendar(calSvc))
	registerServiceTools(server, services.NewGmail(gmailSvc))

	log.Println("✅ Google Services MCP server iniciado (stdio)")
	log.Printf("   Tools registradas: %d\n", len(server.Tools()))

	if err := server.Run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func registerServiceTools(server *mcp.Server, svc services.Service) {
	for _, tool := range svc.Tools() {
		t := tool // capture
		name := t.Name
		server.RegisterTool(name, t, func(ctx context.Context, params json.RawMessage) (interface{}, *mcp.RPCError) {
			return svc.Handle(ctx, name, params)
		})
	}
}

func cmdStatus() {
	token, err := auth.LoadToken()
	if err != nil {
		fmt.Println("⚠ Error al leer token:", err)
		return
	}
	if token == nil {
		fmt.Println("❌ No autenticado.")
		fmt.Println("   Corré: pi-google-services login")
		return
	}
	fmt.Println("✅ Autenticado")
	if !token.Expiry.IsZero() {
		fmt.Printf("  Token expira: %s\n", token.Expiry.Format("2006-01-02 15:04 MST"))
		if token.Expiry.Before(time.Now()) {
			fmt.Println("  ⚠ Token expirado, se renovará al iniciar serve")
		}
	}
	fmt.Printf("\n  Servicios disponibles:\n")
	for _, svc := range registeredServices() {
		fmt.Printf("    • %s (%d tools)\n", svc.Name(), len(svc.Tools()))
	}
	fmt.Printf("\n  Para iniciar: pi-google-services serve\n")
}
