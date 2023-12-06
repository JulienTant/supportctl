package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ensureWorkingDir() (string, error) {
	workDir := viper.GetString("work-dir")
	info, err := os.Stat(workDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat work-dir: %w", err)
		}

		err := os.MkdirAll(workDir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create work-dir folder: %w", err)
		}
	} else if !info.IsDir() {
		return "", fmt.Errorf("work dir is not a folder")
	}

	return workDir, nil
}

func getTicketFolderPath(ticketNumber string) string {
	folderPath := filepath.Join(viper.GetString("work-dir"), "ZD-"+ticketNumber)
	path, err := filepath.Abs(folderPath)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path: %w", err))
	}

	return path
}

func ticketFolderExists(ticketNumber string) bool {
	folderPath := getTicketFolderPath(ticketNumber)
	info, err := os.Stat(folderPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func makeTicketFolderIfNeeded(ticketNumber string) (string, error) {
	_, err := ensureWorkingDir()
	if err != nil {
		return "", fmt.Errorf("failed to ensure working dir: %w", err)
	}

	folderPath := getTicketFolderPath(ticketNumber)
	if err = os.MkdirAll(folderPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create folder: %w", err)
	}

	path, err := filepath.Abs(folderPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return path, nil
}

func mustHaveZendeskConfig(_ *cobra.Command, _ []string) error {
	if viper.GetString("zendesk.subdomain") == "" {
		return fmt.Errorf("zendesk.subdomain is not set")
	}

	if viper.GetString("zendesk.bearer-token") == "" {
		return fmt.Errorf("zendesk.bearer-token is not set")
	}

	return nil
}
