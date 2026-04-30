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
	"github.com/elpdev/telex-cli/internal/calendarstore"
	"github.com/elpdev/telex-cli/internal/config"
	"github.com/elpdev/telex-cli/internal/contactstore"
	"github.com/elpdev/telex-cli/internal/drivestore"
	"github.com/elpdev/telex-cli/internal/drivesync"
	"github.com/elpdev/telex-cli/internal/notestore"
	"github.com/elpdev/telex-cli/internal/taskstore"
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
	cmd.AddCommand(newTasksCommand(rt))
	cmd.AddCommand(newContactsCommand(rt))
	return cmd
}

func newSyncCommand(rt *runtime) *cobra.Command {
	var mailboxAddress string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync all local Telex data",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFullSync(cmd, rt, mailboxAddress)
		},
	}
	cmd.Flags().StringVar(&mailboxAddress, "mailbox", "", "limit mail sync to one synced mailbox address")
	return cmd
}

func runFullSync(cmd *cobra.Command, rt *runtime, mailboxAddress string) error {
	if err := runMailSync(cmd, rt, mailboxAddress); err != nil {
		return fmt.Errorf("mail sync: %w", err)
	}

	service, cfg, err := driveService(rt)
	if err != nil {
		return fmt.Errorf("drive sync: %w", err)
	}
	driveResult, err := drivesync.Run(rt.context(), drivestore.New(rt.dataPath), service, cfg.DriveSyncMode())
	if err != nil {
		return fmt.Errorf("drive sync: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d Drive folder(s), %d file(s).\n", driveResult.Folders, driveResult.Files)
	if cfg.DriveSyncMode() == config.DriveSyncFull {
		fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %d Drive file content(s).\n", driveResult.DownloadedFiles)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Drive metadata-only mode: file contents were not downloaded.")
	}
	if driveResult.DownloadFailures > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Skipped %d Drive file download(s) due to remote errors; metadata was still cached.\n", driveResult.DownloadFailures)
	}

	noteSvc, err := notesService(rt)
	if err != nil {
		return fmt.Errorf("notes sync: %w", err)
	}
	notesResult, err := runNotesSync(rt, noteSvc, notestore.New(rt.dataPath))
	if err != nil {
		return fmt.Errorf("notes sync: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d Notes folder(s), %d note(s).\n", notesResult.Folders, notesResult.Notes)

	taskSvc, err := tasksService(rt)
	if err != nil {
		return fmt.Errorf("tasks sync: %w", err)
	}
	tasksResult, err := runTasksSync(rt.context(), taskstore.New(rt.dataPath), taskSvc)
	if err != nil {
		return fmt.Errorf("tasks sync: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d task project(s), %d board(s), %d card(s).\n", tasksResult.Projects, tasksResult.Boards, tasksResult.Cards)

	calendarSvc, err := calendarService(rt)
	if err != nil {
		return fmt.Errorf("calendar sync: %w", err)
	}
	calendarResult, err := runCalendarSync(rt, calendarSvc, calendarstore.New(rt.dataPath), calendarSyncOptions{})
	if err != nil {
		return fmt.Errorf("calendar sync: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d calendar(s), %d event(s), %d occurrence(s).\n", calendarResult.Calendars, calendarResult.Events, calendarResult.Occurrences)

	contactSvc, err := contactsService(rt)
	if err != nil {
		return fmt.Errorf("contacts sync: %w", err)
	}
	contactsResult, err := runContactsSync(rt, contactSvc, contactstore.New(rt.dataPath))
	if err != nil {
		return fmt.Errorf("contacts sync: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Synced %d contact(s), %d contact note(s).\n", contactsResult.Contacts, contactsResult.Notes)

	return nil
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
	program := tea.NewProgram(app.NewWithPaths(meta, configPath, dataPath), tea.WithEnvironment(os.Environ()))
	_, err := program.Run()
	return err
}

func (r *runtime) context() context.Context { return context.Background() }

func (r *runtime) configFiles() (string, string) { return config.Paths(r.configPath) }

func (r *runtime) prefsPath() string { return config.PrefsPathFor(r.configPath) }

func (r *runtime) loadPrefs() (*config.UIPrefs, error) {
	return config.LoadPrefs(r.prefsPath())
}

func (r *runtime) savePrefs(prefs *config.UIPrefs) error {
	return prefs.SaveTo(r.prefsPath())
}

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
