package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/mackeper/m_backuper/internal/config"
	"github.com/mackeper/m_backuper/internal/scanner"
	"github.com/mackeper/m_backuper/internal/state"
)

var globalConfigPath string

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Global flags
	flag.StringVar(&globalConfigPath, "config", "", "Path to config file")
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	switch command {
	case "backup":
		backupCmd(flag.Args()[1:])
	case "status":
		statusCmd(flag.Args()[1:])
	case "config":
		configCmd(flag.Args()[1:])
	case "init":
		initCmd(flag.Args()[1:])
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
	fmt.Println("  m_backuper [-config path] <command> [options]")
	fmt.Println()
	fmt.Println("Global flags:")
	fmt.Println("  -config string    Path to config file (default: ~/.config/m_backuper/config.json)")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  backup    Run backup")
	fmt.Println("  status    Show last backup time, file count")
	fmt.Println("  config    Show current config (merged file + env)")
	fmt.Println("  init      Generate default config file")
}

func loadConfig() (config.Config, error) {
	if globalConfigPath != "" {
		return config.LoadFrom(globalConfigPath)
	}
	return config.Load()
}

func backupCmd(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Show files that would be backed up without copying")
	fs.Parse(args)

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if len(cfg.PathsToBackup) == 0 {
		slog.Warn("no paths configured for backup")
		fmt.Println("Please configure paths to backup in the config file.")
		fmt.Println("Run 'm_backuper init' to create a default config file.")
		return
	}

	if *dryRun {
		slog.Info("running dry-run scan")
		s := scanner.New(cfg.FilesToIgnorePatterns)
		files, err := s.ScanDryRun(cfg.PathsToBackup)
		if err != nil {
			slog.Error("scan failed", "error", err)
			os.Exit(1)
		}
		fmt.Printf("\nFound %d files that would be backed up\n", len(files))
	} else {
		slog.Info("backup: not implemented")
	}
}

func statusCmd(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	fs.Parse(args)

	// Load state
	st, err := state.Load()
	if err != nil {
		slog.Error("failed to load state", "error", err)
		os.Exit(1)
	}

	// Display status
	fmt.Println("Backup Status:")
	fmt.Println()

	if st.LastRun.IsZero() {
		fmt.Println("  Last backup: Never")
	} else {
		fmt.Printf("  Last backup: %s\n", st.LastRun.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("  Files backed up: %d\n", st.FileCount())

	if st.FileCount() > 0 {
		var totalSize int64
		for _, fileState := range st.Files {
			totalSize += fileState.Size
		}
		fmt.Printf("  Total size: %d bytes (%.2f MB)\n", totalSize, float64(totalSize)/(1024*1024))
	}
}

func configCmd(args []string) {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	fs.Parse(args)

	cfg, err := loadConfig()
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
