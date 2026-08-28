package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all config values",
	RunE:  runConfigList,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	RunE:  runConfigPath,
}

var configTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Mark the current directory's .wenmar.yml as trusted",
	RunE:  runConfigTrust,
}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd, configListCmd, configPathCmd, configTrustCmd)
	rootCmd.AddCommand(configCmd)
}

func resolveConfigPath() (string, error) {
	if configPathFlag != "" {
		return configPathFlag, nil
	}
	return config.ConfigPath()
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		return fmt.Errorf("could not read config: %w", err)
	}

	switch args[0] {
	case "token":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.Token)
	case "base_url":
		fmt.Fprintln(cmd.OutOrStdout(), cfg.BaseURL)
	default:
		return fmt.Errorf("unknown config key: %s (supported: token, base_url)", args[0])
	}
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		// Config doesn't exist yet — create a new one
		cfg = &config.Config{}
	}

	switch args[0] {
	case "token":
		cfg.Token = args[1]
	case "base_url":
		cfg.BaseURL = args[1]
	default:
		return fmt.Errorf("unknown config key: %s (supported: token, base_url)", args[0])
	}

	if err := config.SaveTo(path, cfg); err != nil {
		return fmt.Errorf("could not save config: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Set %s in %s\n", args[0], path)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		return fmt.Errorf("could not read config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "token: %s\n", maskToken(cfg.Token))
	fmt.Fprintf(cmd.OutOrStdout(), "base_url: %s\n", cfg.BaseURL)
	fmt.Fprintf(cmd.OutOrStdout(), "path: %s\n", path)
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), path)
	return nil
}

func runConfigTrust(cmd *cobra.Command, args []string) error {
	// Implemented in Part 3 (per-repo config trust model)
	return fmt.Errorf("config trust is not yet implemented")
}
