package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/orc-analytics/core/protobufs/go"

	"github.com/orc-analytics/cli/stub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// subcommands
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)
	stopCmd := flag.NewFlagSet("stop", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	destroyCmd := flag.NewFlagSet("destroy", flag.ExitOnError)
	stubCmd := flag.NewFlagSet("stub", flag.ExitOnError)
	initCmd := flag.NewFlagSet("init", flag.ExitOnError)

	// TODO: make this a `--help` flag that can be used at any point throughout the process
	helpCmd := flag.NewFlagSet("help", flag.ExitOnError)

	// check if a subcommand is provided
	if len(os.Args) < 2 {
		fmt.Println()
		showHelp()
		fmt.Println()
		os.Exit(1)
	}

	// parse the appropriate subcommand
	switch os.Args[1] {

	case "start":
		checkDockerInstalled()

		startCmd.Parse(os.Args[2:])

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

		fmt.Println(renderSuccess("✅ Orca stack started successfully."))
		fmt.Println()

	case "stop":
		checkDockerInstalled()

		stopCmd.Parse(os.Args[2:])

		fmt.Println()
		stopContainers()

		fmt.Println()
		fmt.Println(renderSuccess("✅ All containers stopped."))
		fmt.Println()

	case "status":
		checkDockerInstalled()
		statusCmd.Parse(os.Args[2:])

		fmt.Println()
		showStatus()
		fmt.Println()

	case "destroy":
		checkDockerInstalled()
		destroyCmd.Parse(os.Args[2:])
		fmt.Println()
		destroy()
		fmt.Println()

	case "init":
		type OrcaConfigFile struct {
			ProjectName          string `json:"projectName"`
			OrcaConnectionString string `json:"connectionString"`
			ProcessorPort        int    `json:"processorPort"`
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
		projectNameFlag := initCmd.String("name", "", "The name of the SDKs repository. Advanced ML")
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

			ProjectName:          projectName,
			OrcaConnectionString: fmt.Sprintf("localhost:%s", orcaPort),
			ProcessorPort:        processorPort,
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
				existingConfig.ProcessorPort != newConfig.ProcessorPort || existingConfig.ProjectName != newConfig.ProjectName {
				fmt.Println("Existing orca.json found with different configuration:")
				fmt.Printf("  Current - Connection: %s, Port: %d, Name: %s\n", existingConfig.OrcaConnectionString, existingConfig.ProcessorPort, existingConfig.ProjectName)
				fmt.Printf("  New     - Connection: %s, Port: %d, Name: %s\n", newConfig.OrcaConnectionString, newConfig.ProcessorPort, newConfig.ProjectName)
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

		data, err := json.Marshal(&newConfig)
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to marshal configuration: %v", err)))
			os.Exit(1)
		}

		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to write orca.json: %v", err)))
			os.Exit(1)
		}

		fmt.Println("orca.json created successfully!")
		fmt.Printf("Project Name: %s\n", newConfig.ProjectName)
		fmt.Printf("Connection String: %s\n", newConfig.OrcaConnectionString)
		fmt.Printf("Processor Port: %d\n", newConfig.ProcessorPort)

	case "stub":
		outDir := stubCmd.String("out", "./orca-stubs", "Output directory for generated stubs")
		orcaConnStr := stubCmd.String("connStr", "", "Orca connection string")

		args := os.Args[2:]
		sdkIndex := -1
		for i, arg := range args {
			if !strings.HasPrefix(arg, "-") {
				sdkIndex = i
				break
			}
		}

		if sdkIndex == -1 {
			fmt.Println(renderError("The SDK target needs to be provided (e.g. python, go, js)"))
			os.Exit(1)
		}

		sdkLanguage := args[sdkIndex]

		// remove the SDK language from args and parse the rest as flags
		flagArgs := append(args[:sdkIndex], args[sdkIndex+1:]...)
		stubCmd.Parse(flagArgs)

		validSDKs := map[string]bool{
			"python": true,
		}

		if !validSDKs[sdkLanguage] {
			fmt.Println(renderError(fmt.Sprintf("Unsupported SDK language: %s", sdkLanguage)))
			fmt.Println(renderInfo("Supported languages: python"))
			os.Exit(1)
		}

		var connStr string
		if *orcaConnStr == "" {
			orcaStatus := getContainerStatus(orcaContainerName)

			if orcaStatus == "running" {
				orcaPort := getContainerPort(orcaContainerName, 3335)
				connStr = fmt.Sprintf("localhost:%s", orcaPort)
			}
		} else {
			connStr = *orcaConnStr
		}

		fmt.Println()
		fmt.Printf("Generating %s stubs to %s...\n", sdkLanguage, *outDir)

		if err := os.MkdirAll(*outDir, 0755); err != nil {
			fmt.Println(renderError(fmt.Sprintf("Failed to create output directory: %v", err)))
			os.Exit(1)
		}
		conn, err := grpc.NewClient(connStr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		defer conn.Close()
		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Issue preparping to contact Orca: %v", err)))
			os.Exit(1)
		}

		orcaCoreClient := pb.NewOrcaCoreClient(conn)
		internalState, err := orcaCoreClient.Expose(context.Background(), &pb.ExposeSettings{})

		if err != nil {
			fmt.Println(renderError(fmt.Sprintf("Issue contacting Orca: %v", err)))
			os.Exit(1)
		}

		if sdkLanguage == "python" {
			err := stub.GeneratePythonStub(internalState, *outDir)
			if err != nil {
				fmt.Println(renderError(fmt.Sprintf("Failed to generate python stubs: %v", err)))
				os.Exit(1)
			}
		}
		fmt.Println(renderSuccess(fmt.Sprintf("✅ %s stubs generated successfully in %s", sdkLanguage, *outDir)))
		fmt.Println()

	case "help":
		helpCmd.Parse(os.Args[2:])
		fmt.Println()
		if helpCmd.NArg() > 0 {
			showCommandHelp(os.Args[2])
		} else {
			showHelp()
		}
		fmt.Println()

	default:
		fmt.Println()
		fmt.Println(renderError(fmt.Sprintf("Unknown subcommand: %s", os.Args[1])))
		fmt.Println(renderInfo("Run 'help' for usage information."))
		fmt.Println()
		os.Exit(1)
	}
}
