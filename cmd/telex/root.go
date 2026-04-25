package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/elpdev/telex-cli/internal/api"
	"github.com/elpdev/telex-cli/internal/app"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/spf13/cobra"
)

type buildInfo = app.BuildInfo

type runtime struct {
	meta       buildInfo
	configPath string
	dataPath   string
	format     string
	client     *api.Client
}

func Execute(meta buildInfo) {
	cmd := newRootCommand(meta)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand(meta buildInfo) *cobra.Command {
	rt := &runtime{meta: meta, format: "table"}
	cmd := &cobra.Command{
		Use:           "telex",
		Short:         "Telex terminal client",
		Version:       fmt.Sprintf("%s (%s, %s)", meta.Version, meta.Commit, meta.Date),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(meta, rt.configPath, rt.dataPath)
		},
	}
	cmd.PersistentFlags().StringVar(&rt.configPath, "config", "", "config file path")
	cmd.PersistentFlags().StringVar(&rt.dataPath, "data-dir", "", "local data directory")
	cmd.PersistentFlags().StringVarP(&rt.format, "format", "f", "table", "output format: table, json, text")
	cmd.AddCommand(newTUICommand(rt))
	cmd.AddCommand(newSyncCommand(rt))
	cmd.AddCommand(newAccountCommand(rt))
	cmd.AddCommand(newMailCommand(rt))
	cmd.AddCommand(newCalendarCommand(rt))
	cmd.AddCommand(newDriveCommand(rt))
	cmd.AddCommand(newNotesCommand(rt))
	return cmd
}

func newSyncCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync local Telex data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMailSync(cmd, rt, mailboxAddress)
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "limit mail sync to one synced mailbox address")
	return cmd
}

func newTUICommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the full-screen TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(rt.meta, rt.configPath, rt.dataPath)
		},
	}
}

func runTUI(meta buildInfo, configPath, dataPath string) error {
	program := tea.NewProgram(app.NewWithPaths(meta, configPath, dataPath))
	_, err := program.Run()
	return err
}

func (r *runtime) context() context.Context { return context.Background() }

func (r *runtime) configFiles() (string, string) { return config.Paths(r.configPath) }

func (r *runtime) loadConfig() (*config.Config, error) {
	configFile, _ := r.configFiles()
	cfg, err := config.LoadFrom(configFile)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (r *runtime) apiClient() (*api.Client, error) {
	if r.client != nil {
		return r.client, nil
	}
	cfg, err := r.loadConfig()
	if err != nil {
		return nil, err
	}
	_, tokenFile := r.configFiles()
	r.client = api.NewClient(cfg, tokenFile)
	return r.client, nil
}

func writeRows(w io.Writer, headers []string, rows [][]string) {
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
}
