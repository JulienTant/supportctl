package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/julientant/supportctl/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// pruneCmd represents the prune command
var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "remove old ticket folders",
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir, err := ensureWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to ensure working dir: %w", err)
		}

		zd, err := zendesk.NewClientFromViper(viper.GetViper())
		if err != nil {
			return fmt.Errorf("failed to create zendesk client: %w", err)
		}

		ticketsToCheck := []int64{}
		filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("failed to walk: %w", err)
			}

			if info.IsDir() && info.Name() != workDir {
				if strings.HasPrefix(info.Name(), "ZD-") {
					if _, err := os.Stat(filepath.Join(path, ".supportctl-keep")); err == nil {
						return nil
					}

					idStr := strings.TrimPrefix(info.Name(), "ZD-")
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err != nil {
						return fmt.Errorf("failed to parse ticket id: %w", err)
					}
					ticketsToCheck = append(ticketsToCheck, id)
				}
			}
			return nil
		})

		if len(ticketsToCheck) == 0 {
			log.Println("No tickets to check")
			return nil
		}

		// create chunks of 100 tickets
		chunks := [][]int64{}
		chunkSize := 100
		for i := 0; i < len(ticketsToCheck); i += chunkSize {
			end := i + chunkSize
			if end > len(ticketsToCheck) {
				end = len(ticketsToCheck)
			}
			chunks = append(chunks, ticketsToCheck[i:end])
		}

		closedSinceInDays := viper.GetInt("prune.closed-since-days")
		if closedSinceInDays <= 0 {
			log.Println("prune.closed-since-days must be greater than 0, using 30 days")
			closedSinceInDays = 30
		}

		for _, chunk := range chunks {
			tickets, err := zd.GetMultipleTickets(cmd.Context(), chunk)
			if err != nil {
				return fmt.Errorf("failed to retrieve tickets: %w", err)
			}

			for _, ticket := range tickets {
				switch ticket.Status {
				case "solved", "closed":
					if ticket.UpdatedAt.AddDate(0, 0, closedSinceInDays).Before(time.Now()) {
						log.Printf("Removing folder for ticket %d\n", ticket.ID)
						if err := os.RemoveAll(getTicketFolderPath(strconv.FormatInt(ticket.ID, 10))); err != nil {
							return fmt.Errorf("failed to remove folder: %w", err)
						}
					}
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pruneCmd)

	pruneCmd.Flags().Int("prune.closed-since-days", 30, "removed tickets folder that have been closed for more than this number of days")
	viper.BindPFlag("prune.closed-since-days", pruneCmd.Flags().Lookup("prune.closed-since-days"))
}
