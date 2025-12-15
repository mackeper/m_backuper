package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mackeper/m_backuper/internal/config"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "backup":
		backupCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	case "config":
		configCmd(os.Args[2:])
	case "init":
		initCmd(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("m_backuper - Incremental backup tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  m_backuper <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  backup    Run backup")
	fmt.Println("  status    Show last backup time, file count")
	fmt.Println("  config    Show current config (merged file + env)")
	fmt.Println("  init      Generate default config file")
}

func backupCmd(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Show files that would be backed up without copying")
	fs.Parse(args)

	if *dryRun {
		slog.Info("backup: dry-run not implemented")
	} else {
		slog.Info("backup: not implemented")
	}
}

func statusCmd(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Parse(args)

	slog.Info("status: not implemented")
}

func configCmd(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Parse(args)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	fmt.Println(cfg.String())
}

func initCmd(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	fs.Parse(args)

	cfg := config.Default()
	if err := config.Save(cfg); err != nil {
		slog.Error("failed to save config", "error", err)
		os.Exit(1)
	}

	configPath, _ := config.ConfigPath()
	fmt.Printf("Created default config at: %s\n", configPath)
	fmt.Println("\nEdit this file to configure your backup settings.")
}
