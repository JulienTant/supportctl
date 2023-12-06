package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// keepCmd represents the keep command
var keepCmd = &cobra.Command{
	Use:       "keep [ticket number]",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"ticket number"},
	Short:     "Make sure prune does not remove this ticket folder",
	RunE: func(cmd *cobra.Command, args []string) error {
		ticketNumber := args[0]

		if !ticketFolderExists(ticketNumber) {
			return errors.New("ticket folder does not exist")
		}

		// create a .supportctl-keep file
		keepFilePath := filepath.Join(getTicketFolderPath(ticketNumber), ".supportctl-keep")
		f, err := os.Create(keepFilePath)
		if err != nil {
			return fmt.Errorf("failed to create keep file: %w", err)
		}
		defer f.Close()

		log.Printf("Created keep file: %s for ticket %s", keepFilePath, ticketNumber)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(keepCmd)
}
