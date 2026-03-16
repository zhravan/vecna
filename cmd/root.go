package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/shravan20/vecna/internal/config"
	"github.com/shravan20/vecna/internal/tui"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	cfgFile string
)

func init() {
	if Version == "dev" {
		if b, err := os.ReadFile("version.txt"); err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				Version = v
			}
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "vecna",
	Short: "SSH manager TUI",
	Long:  "Vecna - A minimalist SSH manager with TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run(Version)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.config/vecna/config.yaml)")
}

func initConfig() {
	config.Init(cfgFile)
}
