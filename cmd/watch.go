package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/julientant/supportctl/zendesk"
	zdlib "github.com/nukosuke/go-zendesk/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// watchCmd represents the ui command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Start a view watcher",
	RunE: func(cmd *cobra.Command, args []string) error {
		zd, err := zendesk.NewClientFromViper(viper.GetViper())
		if err != nil {
			return fmt.Errorf("failed to create zendesk client: %w", err)
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill)
		defer stop()

		var viewID int64
		viewName := viper.GetString("watch.view")
		views, _, err := zd.GetViews(ctx)
		if err != nil {
			return fmt.Errorf("failed to get views: %w", err)
		}
		for _, v := range views {
			if v.Title == viewName {
				viewID = v.ID
				break
			}
		}

		ticket := time.NewTicker(time.Minute * time.Duration(viper.GetInt64("watch.frequency")))
		for {
			pageOptions := zdlib.PageOptions{PerPage: 100, Page: 1}
			allTickets := []zdlib.Ticket{}
			for {
				tickets, page, err := zd.GetTicketsFromView(ctx, viewID, &zdlib.TicketListOptions{
					PageOptions: pageOptions,
				})
				if err != nil {
					return fmt.Errorf("failed to search tickets: %w", err)
				}

				for _, ticket := range tickets {
					allTickets = append(allTickets, ticket)
				}

				if page.HasNext() {
					pageOptions.Page++
				} else {
					break
				}
			}

			if len(allTickets) > 0 {
				if err := beeep.Alert("SupportCTL", fmt.Sprintf("You have %d tickets in the queue", len(allTickets)), "assets/information.png"); err != nil {
					return fmt.Errorf("failed to send notification: %w", err)
				}
			}

			select {
			case <-ticket.C:
				continue
			case <-ctx.Done():
				log.Println("Stopping watcher")
				return nil
			}

		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	watchCmd.Flags().String("watch.view", "Support - New & Unassigned", "View to watch")
	viper.BindPFlag("watch.view", watchCmd.Flags().Lookup("watch.view"))

	watchCmd.Flags().Int64("watch.frequency", 1, "Frequency in minutes")
	viper.BindPFlag("watch.frequency", watchCmd.Flags().Lookup("watch.frequency"))

}
