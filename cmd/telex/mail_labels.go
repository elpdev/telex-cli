package main

import "github.com/spf13/cobra"

func newLabelsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "labels", Short: "Manage remote labels"}
	cmd.AddCommand(newLabelsListCommand(rt))
	return cmd
}

func newLabelsListCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List remote labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			labels, err := service.Labels(rt.context())
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(labels))
			for _, label := range labels {
				rows = append(rows, labelRow(label))
			}
			writeRows(cmd.OutOrStdout(), []string{"id", "name", "color"}, rows)
			return nil
		},
	}
}
