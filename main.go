package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/orc-analytics/cli/stub"
	pb "github.com/orc-analytics/core/protobufs/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Orca CLI\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  orca <command> [options]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  start    Start the Orca stack\n")
		fmt.Fprintf(os.Stderr, "  stop     Stop all Orca containers\n")
		fmt.Fprintf(os.Stderr, "  status   Show status of Orca components\n")
		fmt.Fprintf(os.Stderr, "  destroy  Delete all Orca resources\n")
		fmt.Fprintf(os.Stderr, "  init     Initialize orca.json configuration\n")
		fmt.Fprintf(os.Stderr, "  sync     Sync Orca registry data\n")
		fmt.Fprintf(os.Stderr, "  help     Show help information\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  orca start\n")
		fmt.Fprintf(os.Stderr, "  orca sync -out ./data\n")
		fmt.Fprintf(os.Stderr, "  orca init -name myproject\n\n")
		fmt.Fprintf(os.Stderr, "For more information on a command, run:\n")
		fmt.Fprintf(os.Stderr, "  orca <command> help / -h\n")
		flag.PrintDefaults()
	}

	// subcommands
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	destroyCmd := flag.NewFlagSet("destroy", flag.ExitOnError)
	syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	// check if a subcommand is provided
	if len(os.Args) < 2 {
		fmt.Println()
		flag.Usage()
		fmt.Println()
		os.Exit(1)
	}

	// parse the appropriate subcommand
	switch os.Args[1] {

	case "start":
		startCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: orca start\n\n")
			fmt.Fprintf(os.Stderr, "Start the Orca stack (Postgres, Redis, and Orca services)\n")
		}

		startCmd.Parse(os.Args[2:])

		if startCmd.NArg() > 0 && (startCmd.Arg(0) == "help" || startCmd.Arg(0) == "-h") {
			startCmd.Usage()
			os.Exit(0)
		}

		if startCmd.NArg() > 0 {
			fmt.Println()
			fmt.Println(renderError(fmt.Sprintf("Unknown argument: %s", startCmd.Arg(0))))
			fmt.Println("Run 'orca start help' for usage information.")
			fmt.Println()
			os.Exit(1)
		}

		checkDockerInstalled()

		fmt.Println()
		networkName := createNetworkIfNotExists()
		fmt.Println()

		startPostgres(networkName)
		fmt.Println()

		startRedis(networkName)
		fmt.Println()

		// check for postgres instance running first
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		err := waitForPgReady(ctx, pgContainerName, time.Millisecond*500)
		if err != nil {
			fmt.Println(
				renderError(
					fmt.Sprintf("Issue waiting for Postgres store to start: %v", err.Error()),
				),
			)
			os.Exit(1)
		}
		startOrca(networkName)
		fmt.Println()

		fmt.Println(renderSuccess(" Orca stack started successfully."))
		fmt.Println()

	case "stop":
		stopCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: orca stop\n\n")
			fmt.Fprintf(os.Stderr, "Stop all running Orca containers\n")
		}

		stopCmd.Parse(os.Args[2:])

		if stopCmd.NArg() > 0 && (stopCmd.Arg(0) == "help" || stopCmd.Arg(0) == "-h") {
			stopCmd.Usage()
			os.Exit(0)
		}

		if stopCmd.NArg() > 0 {
			fmt.Println()
			fmt.Println(renderError(fmt.Sprintf("Unknown argument: %s", stopCmd.Arg(0))))
			fmt.Println("Run 'orca stop help' for usage information.")
			fmt.Println()
			os.Exit(1)
		}

		checkDockerInstalled()

		fmt.Println()
		stopContainers()

		fmt.Println()
		fmt.Println(renderSuccess(" All containers stopped."))
		fmt.Println()

	case "status":
		statusCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: orca status\n\n")
			fmt.Fprintf(os.Stderr, "Show the status of all Orca components\n")
		}

		statusCmd.Parse(os.Args[2:])

		if statusCmd.NArg() > 0 && (statusCmd.Arg(0) == "help" || statusCmd.Arg(0) == "-h") {
			statusCmd.Usage()
			os.Exit(0)
		}

		if statusCmd.NArg() > 0 {
			fmt.Println()
			fmt.Println(renderError(fmt.Sprintf("Unknown argument: %s", statusCmd.Arg(0))))
			fmt.Println("Run 'orca status help' for usage information.")
			fmt.Println()
			os.Exit(1)
		}

		checkDockerInstalled()

		fmt.Println()
		showStatus()
		fmt.Println()

	case "destroy":
		destroyCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: orca destroy\n\n")
			fmt.Fprintf(os.Stderr, "Delete all Orca resources (containers, volumes, networks)\n")
		}

		destroyCmd.Parse(os.Args[2:])

		if destroyCmd.NArg() > 0 && (destroyCmd.Arg(0) == "help" || destroyCmd.Arg(0) == "-h") {
			destroyCmd.Usage()
			os.Exit(0)
		}

		if destroyCmd.NArg() > 0 {
			fmt.Println()
			fmt.Println(renderError(fmt.Sprintf("Unknown argument: %s", destroyCmd.Arg(0))))
			fmt.Println("Run 'orca destroy help' for usage information.")
			fmt.Println()
			os.Exit(1)
		}

		checkDockerInstalled()
		fmt.Println()
		destroy()
		fmt.Println()

	case "init":
		projectNameFlag := initCmd.String("name", "", "Project name (defaults to current directory name)")

		initCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: orca init [options]\n\n")
			fmt.Fprintf(os.Stderr, "Initialise orca.json configuration file\n\n")
			fmt.Fprintf(os.Stderr, "Options:\n")
			initCmd.PrintDefaults()
		}

		initCmd.Parse(os.Args[2:])

		if initCmd.NArg() > 0 && (initCmd.Arg(0) == "help" || initCmd.Arg(0) == "-h") {
			initCmd.Usage()
			os.Exit(0)
		}

		if initCmd.NArg() > 0 {
			fmt.Println()
			fmt.Println(renderError(fmt.Sprintf("Unknown argument: %s", initCmd.Arg(0))))
			fmt.Println("Run 'orca init help' for usage information.")
			fmt.Println()
			os.Exit(1)
		}

		type OrcaConfigFile struct {
			ProjectName               string `json:"projectName"`
			OrcaConnectionString      string `json:"connectionString"`
			ProcessorPort             int    `json:"processorPort"`
			ProcessorConnectionString string `json:"processorConnectionString"`
		}
		preferredProcessorPort := 5377

		orcaStatus := getContainerStatus(orcaContainerName)
		if orcaStatus != "running" {
			fmt.Println(renderError("Orca not running. Cannot initialise configuration file. Start orca locally with the command `orca start`"))
			os.Exit(1)
		}

		orcaPort := getContainerPort(orcaContainerName, orcaInternalPort)
		processorPort := findAvailablePort(preferredProcessorPort)

		if processorPort < 0 {
			fmt.Println(renderError("Could not find an available port to use for the processor"))
			os.Exit(1)
		}
		var projectName string
		if *projectNameFlag != "" {
			projectName = *projectNameFlag
		} else {
			// infer from parent directory name
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Println(renderError(fmt.Sprintf("Failed to get current directory: %v", err)))
				os.Exit(1)
			}
			projectName = toCamelCase(filepath.Base(cwd))
		}

		newConfig := OrcaConfigFile{
			ProjectName:               projectName,
			OrcaConnectionString:      fmt.Sprintf("localhost:%s", orcaPort),
			ProcessorPort:             processorPort,
			ProcessorConnectionString: fmt.Sprintf("host.docker.internal:%d", processorPort),
		}

		configPath := "orca.json"

		if _, err := os.Stat(configPath); err == nil {
			existingData, err := os.ReadFile(configPath)
			if err != nil {
				fmt.Println(renderError(fmt.Sprintf("Failed to read existing orca.json: %v", err)))
				os.Exit(1)
			}

			var existingConfig OrcaConfigFile
			err = json.Unmarshal(existingData, &existingConfig)
			if err != nil {
				fmt.Println(renderError(fmt.Sprintf("Failed to parse existing orca.json: %v", err)))
				os.Exit(1)
			}

			// compare configurations
			if existingConfig.OrcaConnectionString != newConfig.OrcaConnectionString ||
				existingConfig.ProcessorPort != newConfig.ProcessorPort ||
				existingConfig.ProjectName != newConfig.ProjectName ||
				existingConfig.ProcessorConnectionString != newConfig.ProcessorConnectionString {
				fmt.Println("Existing orca.json found with different configuration:")
				fmt.Printf("  Current - Connection: %s, Port: %d, Name: %s, ProcessorConnection: %s\n", existingConfig.OrcaConnectionString, existingConfig.ProcessorPort, existingConfig.ProjectName, existingConfig.ProcessorConnectionString)
				fmt.Printf("  New     - Connection: %s, Port: %d, Name: %s, ProcessorConnection: %s\n", newConfig.OrcaConnectionString, newConfig.ProcessorPort, newConfig.ProjectName, newConfig.ProcessorConnectionString)
				fmt.Print("Do you want to update the configuration? (y/n): ")

				var response string
				fmt.Scanln(&response)

				if strings.ToLower(strings.TrimSpace(response)) != "y" {
					fmt.Println("Configuration update cancelled.")
					os.Exit(0)
				}
			} else {
				fmt.Println("Existing orca.json matches current configuration. No update needed.")
				os.Exit(0)
			}
		}

		data, err := json.MarshalIndent(&newConfig, "", "    ")
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to marshal configuration: %v", err)))
			os.Exit(1)
		}

		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to write orca.json: %v", err)))
			os.Exit(1)
		}

		fmt.Println(successStyle.Render("orca.json created successfully!"))
		fmt.Printf("Project name: %s\n", newConfig.ProjectName)
		fmt.Printf("Orca connection string: %s\n", newConfig.OrcaConnectionString)
		fmt.Printf("Processor port: %d\n", newConfig.ProcessorPort)
		fmt.Printf("Processor connection string: %s\n", newConfig.ProcessorConnectionString)

	case "sync":
		outDir := syncCmd.String("out", ".orca", "Output directory for Orca registry data")
		orcaConnStr := syncCmd.String("connStr", "", "Orca connection string (defaults to local Orca)")
		tgtSdk := syncCmd.String("sdk", "", "The SDK to generate type stubs for - python|go|typescript|zig|rust (defaults to inferring from the environment)")
		secure := syncCmd.Bool("secure", false, "Set to connect to Orca core with System Default Root CA credentials (via TLS). Only use when using a custom Orca connection string that supports TLS")
		caCert := syncCmd.String("caCert", "", "Path to custom CA certificate file (PEM format) for TLS verification")

		syncCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: orca sync [options]\n\n")
			fmt.Fprintf(os.Stderr, "Sync Orca registry data to local directory\n\n")
			fmt.Fprintf(os.Stderr, "Options:\n")
			syncCmd.PrintDefaults()
		}

		syncCmd.Parse(os.Args[2:])

		if syncCmd.NArg() > 0 && (syncCmd.Arg(0) == "help" || syncCmd.Arg(0) == "-h") {
			syncCmd.Usage()
			os.Exit(0)
		}

		if syncCmd.NArg() > 0 {
			fmt.Println()
			fmt.Println(renderError(fmt.Sprintf("Unknown argument: %s", syncCmd.Arg(0))))
			fmt.Println("Run 'orca sync help' for usage information.")
			fmt.Println()
			os.Exit(1)
		}

		type SDKType string

		const (
			SDKPython     SDKType = "python"
			SDKGo         SDKType = "go"
			SDKTypeScript SDKType = "typescript"
			SDKZig        SDKType = "zig"
			SDKRust       SDKType = "rust"
		)

		var validSDKs = map[SDKType]bool{
			SDKPython:     true,
			SDKGo:         false,
			SDKTypeScript: false,
			SDKZig:        false,
			SDKRust:       false,
		}

		if *tgtSdk != "" {
			if !validSDKs[SDKType(*tgtSdk)] {
				fmt.Println(renderError(fmt.Sprintf("Invalid SDK: %s. Must be one of: python, go, typescript, zig, rust\n", *tgtSdk)))
				os.Exit(1)
			}

		} else {
			// Python detection
			if _, err := os.Stat("./pyproject.toml"); !os.IsNotExist(err) {
				*tgtSdk = "python"
			} else if _, err := os.Stat("./requirements.txt"); !os.IsNotExist(err) {
				*tgtSdk = "python"
			} else if _, err := os.Stat("./setup.py"); !os.IsNotExist(err) {
				*tgtSdk = "python"
			} else if _, err := os.Stat("./setup.cfg"); !os.IsNotExist(err) {
				*tgtSdk = "python"
			} else if _, err := os.Stat("./Pipfile"); !os.IsNotExist(err) {
				*tgtSdk = "python"
				// 	// Go detection
				// } else if _, err := os.Stat("./go.mod"); !os.IsNotExist(err) {
				// 	*tgtSdk = "go"
				//
				// 	// TypeScript/JavaScript detection
				// } else if _, err := os.Stat("./package.json"); !os.IsNotExist(err) {
				// 	*tgtSdk = "typescript"
				// } else if _, err := os.Stat("./tsconfig.json"); !os.IsNotExist(err) {
				// 	*tgtSdk = "typescript"
				//
				// 	// Zig detection
				// } else if _, err := os.Stat("./build.zig"); !os.IsNotExist(err) {
				// 	*tgtSdk = "zig"
				//
				// 	// Rust detection
				// } else if _, err := os.Stat("./Cargo.toml"); !os.IsNotExist(err) {
				// 	*tgtSdk = "rust"
			} else {
				fmt.Println(renderError("Cannot infer language from environment. Specify it with the `sdk` command. Run `orca sync help` for more information"))
				os.Exit(1)
			}
			fmt.Printf("Inferred sdk langauge as %v\n", *tgtSdk)
		}

		var connStr string
		if *orcaConnStr == "" {
			orcaStatus := getContainerStatus(orcaContainerName)

			if orcaStatus == "running" {
				orcaPort := getContainerPort(orcaContainerName, 3335)
				connStr = fmt.Sprintf("localhost:%s", orcaPort)
			} else {
				fmt.Println(renderError("Orca is not running. Cannot generate registry data. Start Orca with `orca start`"))
				os.Exit(1)
			}
		} else {
			connStr = *orcaConnStr
		}

		fmt.Printf("Generating registry data to %s\n", *outDir)

		if err := os.MkdirAll(*outDir, 0755); err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to create output directory: %v", err)))
			os.Exit(1)
		}
		var conn *grpc.ClientConn
		var err error
		var transportCreds credentials.TransportCredentials

		if *caCert != "" {
			// user provided a specific CA file
			pemServerCA, err := os.ReadFile(*caCert)
			if err != nil {
				fmt.Println(renderError(fmt.Sprintf("Failed to read CA certificate: %v", err)))
				os.Exit(1)
			}

			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(pemServerCA) {
				fmt.Println(renderError("Failed to add CA certificate to pool (invalid PEM format?)"))
				os.Exit(1)
			}

			config := &tls.Config{
				RootCAs: certPool,
			}
			transportCreds = credentials.NewTLS(config)
			fmt.Println("Using custom CA certificate for TLS...")

		} else if *secure {
			// use system default certificates
			transportCreds = credentials.NewTLS(&tls.Config{})
			fmt.Println("Using system default CA for TLS...")
		} else {
			// insecure connection - good for accessing internal Orca service
			transportCreds = insecure.NewCredentials()
		}
		conn, err = grpc.NewClient(connStr, grpc.WithTransportCredentials(transportCreds))
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Issue preparing to contact Orca: %v", err)))
			os.Exit(1)
		}
		defer conn.Close()

		orcaCoreClient := pb.NewOrcaCoreClient(conn)
		internalState, err := orcaCoreClient.Expose(context.Background(), &pb.ExposeSettings{})

		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Issue contacting Orca: %v", err)))
			os.Exit(1)
		}
		data, err := json.MarshalIndent(internalState, "", "    ")
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to marshal configuration: %v", err)))
			os.Exit(1)
		}

		err = os.WriteFile(filepath.Join(*outDir, "registry.json"), data, 0644)
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to write orca.json: %v", err)))
			os.Exit(1)
		}

		fmt.Println(renderSuccess(fmt.Sprintf("registry data generated successfully in %s", filepath.Join(*outDir, "registry.json"))))

		switch SDKType(*tgtSdk) {
		case SDKPython:
			fmt.Printf("Generating python stubs to %s\n", *outDir)
			err := stub.GeneratePythonStubs(internalState, *outDir)
			if err != nil {
				fmt.Println(renderError(fmt.Sprintf("Issue generating python stubs: %s", err)))
				os.Exit(1)
			}
			fmt.Println(renderSuccess(fmt.Sprintf("python stubs successfully generated in %s", *outDir)))
		}

	case "help":
		fmt.Println()
		flag.Usage()
		fmt.Println()
		os.Exit(0)
	case "-h":
		fmt.Println()
		flag.Usage()
		fmt.Println()
		os.Exit(0)

	default:
		fmt.Println()
		fmt.Println(renderError(fmt.Sprintf("Unknown subcommand: %s", os.Args[1])))
		fmt.Println("Run 'orca help' for usage information.")
		fmt.Println()
		os.Exit(1)
	}
}
