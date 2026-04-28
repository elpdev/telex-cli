package main

import "github.com/spf13/cobra"

func newConversationsCommand(rt *runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "conversations", Short: "Inspect conversation threads"}
	cmd.AddCommand(newConversationTimelineCommand(rt))
	return cmd
}

func newConversationTimelineCommand(rt *runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "timeline <id>",
		Short: "Show a remote conversation timeline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			service, err := mailService(rt)
			if err != nil {
				return err
			}
			entries, err := service.ConversationTimeline(rt.context(), id)
			if err != nil {
				return err
			}
			rows := make([][]string, 0, len(entries))
			for _, entry := range entries {
				rows = append(rows, conversationTimelineRow(entry))
			}
			writeRows(cmd.OutOrStdout(), []string{"kind", "id", "subject", "sender", "recipients", "status", "occurred_at"}, rows)
			return nil
		},
	}
}
