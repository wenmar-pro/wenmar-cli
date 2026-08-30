package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wenmar-pro/wenmar-cli/internal/config"
	"github.com/wenmar-pro/wenmar-cli/internal/errors"
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

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show resolved config values with their source",
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd, configListCmd, configPathCmd, configTrustCmd, configShowCmd)
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

	fmt.Fprintf(cmd.OutOrStdout(), "token: %s\n", errors.MaskToken(cfg.Token))
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
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	if err := config.TrustRepo(cwd); err != nil {
		return fmt.Errorf("could not trust repo: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Trusted %s. Per-repo .wenmar.yml base_url will now be applied.\n", cwd)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath()
	if err != nil {
		return err
	}

	values := config.ResolveWithProvenance(tokenFlag, baseURLFlag, locationFlag, path)

	// Mask the token for display.
	if v, ok := values["token"]; ok && v.Value != "" {
		v.Value = errors.MaskToken(v.Value)
		values["token"] = v
	}

	for _, key := range []string{"token", "base_url", "location_id", "auth_method"} {
		v, ok := values[key]
		if !ok {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%-12s %s  (from: %s)\n", key+":", v.Value, v.Source)
	}
	return nil
}
