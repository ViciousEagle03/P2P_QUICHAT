package cmd

import (
	"os"
	"os/signal"

	"github.com/ViciousEagle03/P2P_QUICHAT/internal/app"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the P2P chat node",
	Long: `Start a chat node that connects over libp2p gossip-sub.
Examples:
  quichat run --listen 4001 --nick alice
  quichat run --listen 4003 --bootstrap /ip4/…/p2p/… --nick bob`,

	// Only define RunE (or Run), not both
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()

		port, _ := cmd.Flags().GetString("listen")
		bootstrap, _ := cmd.Flags().GetString("bootstrap")
		nick, _ := cmd.Flags().GetString("nick")

		node, err := app.NewNode(ctx, nick, port, bootstrap)
		if err != nil {
			return err
		}

		return app.ChatLoop(ctx, node, nick)
	},
}

func init() {
	// Add it exactly once
	rootCmd.AddCommand(runCmd)

	// Flags
	runCmd.Flags().String("listen", "4001", "port to listen on")
	runCmd.Flags().String("bootstrap", "", "multiaddr of a bootstrap peer")
	runCmd.Flags().String("nick", "anon", "display name")
}
