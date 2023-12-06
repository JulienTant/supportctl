/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/julientant/supportctl/zendesk"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to zendesk to retrieve a token",
	RunE: func(cmd *cobra.Command, args []string) error {
		// get subdomain, email
		subdomain := viper.GetString("zendesk.subdomain")

		subdomainPrompt := promptui.Prompt{
			Label:   "Subdomain",
			Default: subdomain,
			Validate: func(input string) error {
				if len(input) == 0 {
					return fmt.Errorf("Subdomain cannot be empty")
				}
				return nil
			},
		}
		subdomain, err := subdomainPrompt.Run()
		if err != nil {
			return fmt.Errorf("Error getting subdomain from prompt: %s \n", err)
		}

		emailPrompt := promptui.Prompt{
			Label: "Email",
			Validate: func(input string) error {
				if len(input) == 0 {
					return fmt.Errorf("Email cannot be empty")
				}

				// loose validation by making sure there's an @
				if !strings.Contains(input, "@") {
					return fmt.Errorf("Email must contain an @")
				}

				return nil
			},
		}
		email, err := emailPrompt.Run()
		if err != nil {
			return fmt.Errorf("Error getting email from prompt: %s \n", err)
		}

		// ask for password
		passwordPrompt := promptui.Prompt{
			Label: "Password",
			Mask:  '*',
			Validate: func(input string) error {
				if len(input) == 0 {
					return fmt.Errorf("Password cannot be empty")
				}
				return nil
			},
		}
		password, err := passwordPrompt.Run()
		if err != nil {
			return fmt.Errorf("Error getting password from prompt: %s \n", err)
		}

		zd, err := zendesk.NewClient(subdomain, "")
		if err != nil {
			return fmt.Errorf("Error creating zendesk client: %s \n", err)
		}

		result, err := zd.GetBearerToken(cmd.Context(), email, password)
		if err != nil {
			return fmt.Errorf("Error getting bearer token: %s \n", err)
		}

		viper.Set("zendesk.subdomain", subdomain)
		viper.Set("zendesk.bearer-token", result)

		err = viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("Error writing config: %s \n", err)
		}

		fmt.Println("Subdomain and Token saved to config file")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loginCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loginCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
