package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/julientant/supportctl/filedownloader"
	"github.com/julientant/supportctl/git"
	"github.com/julientant/supportctl/zendesk"
	zdlib "github.com/nukosuke/go-zendesk/zendesk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// find the server_version from latest-support-packet/support_packet.yaml
type supportPacket struct {
	ServerVersion string `yaml:"server_version"`
}

var supportPacketRegex = regexp.MustCompile(`mattermost_support_packet_\d{4}-\d{2}-\d{2}-\d{2}-\d{2}.zip`)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:       "get [ticket number]",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"ticket number"},
	Short:     "make a local environment for this ticket",
	PreRunE:   mustHaveZendeskConfig,
	RunE: func(cmd *cobra.Command, args []string) error {
		// ticket number from args
		ticketNumberStr := args[0]
		ticketNumber, err := strconv.ParseInt(ticketNumberStr, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse ticket number: %w", err)
		}

		zd, err := zendesk.NewClientFromViper(viper.GetViper())
		if err != nil {
			return fmt.Errorf("failed to create zendesk client: %w", err)
		}

		allAttachments := viper.GetBool("get.all-attachments")

		ticket, err := zd.GetTicket(cmd.Context(), ticketNumber)
		if err != nil {
			return fmt.Errorf("failed to retrieve ticket: %w", err)
		}
		log.Printf("Found ticket: %s\n", ticket.Subject)

		folder, err := makeTicketFolderIfNeeded(ticketNumberStr)
		if err != nil {
			return fmt.Errorf("failed to create ticket folder: %w", err)
		}

		// check the ticket for support packet
		toDownloadFilesNames := []string{}
		toDownloadedMap := map[string]string{}
		commentQuery := &zdlib.ListTicketCommentsOptions{
			CursorPagination: zdlib.CursorPagination{
				PageSize:  100,
				PageAfter: "",
			},
		}
		for {
			commentsRes, err := zd.ListTicketComments(cmd.Context(), ticketNumber, commentQuery)
			if err != nil {
				return fmt.Errorf("failed to retrieve ticket comments: %w", err)
			}
			for _, comment := range commentsRes.TicketComments {
				for _, a := range comment.Attachments {
					if allAttachments || supportPacketRegex.MatchString(a.FileName) {
						if _, ok := toDownloadedMap[a.FileName]; ok {
							continue
						}
						toDownloadFilesNames = append(toDownloadFilesNames, a.FileName)
						toDownloadedMap[a.FileName] = a.ContentURL
					}
				}
			}
			if !commentsRes.Meta.HasMore {
				break
			}
			log.Println("Going for the next page")
			commentQuery.PageAfter = commentsRes.Meta.AfterCursor
		}

		// filenames matches the regex, we should extract the date from the filename
		// and sort them by date descending
		sort.Slice(toDownloadFilesNames, func(i, j int) bool {
			return toDownloadFilesNames[i] > toDownloadFilesNames[j]
		})

		// download the attachments
		latestSupportPacket := ""
		for _, fileName := range toDownloadFilesNames {
			if !allAttachments && !supportPacketRegex.MatchString(fileName) {
				continue
			}

			log.Printf("Downloading attachment %s to %s\n", fileName, folder)

			var f filedownloader.File
			// for now we only support http get file
			f = filedownloader.NewHTTPGetFile(toDownloadedMap[fileName])
			filePath := filepath.Join(folder, fileName)
			err := f.Download(filePath)
			if err != nil {
				return fmt.Errorf("failed to download support packet: %w", err)
			}

			if latestSupportPacket == "" && supportPacketRegex.MatchString(fileName) {
				latestSupportPacket = fileName
			}
			if latestSupportPacket != "" && !allAttachments {
				break
			}
		}

		// cloning  in the folder
		csReproDest := path.Join(folder, "cs-repro")
		_, err = os.Stat(csReproDest)
		if os.IsNotExist(err) {
			gitClient := git.NewLibClient()

			log.Println("Cloning CS-Repro-Mattermost")
			err = gitClient.Clone(cmd.Context(), viper.GetString("get.cs-repro-repo"), csReproDest)
			if err != nil {
				return fmt.Errorf("failed to clone repo: %w", err)
			}

			log.Println("Renaming containers to add ticket number")
			err = replaceInFolder(csReproDest, "cs-repro-", "cs-repro-"+ticketNumberStr+"-")
			if err != nil {
				return fmt.Errorf("failed to replace in folder: %w", err)
			}
		}

		var sp supportPacket
		if latestSupportPacket != "" {
			log.Println("Support packet found")

			// removing existing latest-support-packet folder
			latestSupportPacketFolder := path.Join(folder, "latest-support-packet")
			_, err = os.Stat(latestSupportPacketFolder)
			if !os.IsNotExist(err) {
				log.Printf("Removing %s\n", latestSupportPacketFolder)
				err = os.RemoveAll(latestSupportPacketFolder)
				if err != nil {
					return fmt.Errorf("failed to remove latest-support-packet folder: %w", err)
				}
			}

			// unzip the support packet in a new folder called latest-support-packet
			log.Printf("Unzipping %s\n", latestSupportPacket)
			archive, err := zip.OpenReader(path.Join(folder, latestSupportPacket))
			if err != nil {
				return fmt.Errorf("failed to open zip file: %w", err)
			}
			defer archive.Close()

			for _, f := range archive.File {
				filePath := filepath.Join(latestSupportPacketFolder, f.Name)

				if f.FileInfo().IsDir() {
					err = os.MkdirAll(filePath, os.ModePerm)
					if err != nil {
						return fmt.Errorf("failed to create directory: %w", err)
					}
					continue
				}

				if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}

				outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}

				rc, err := f.Open()
				if err != nil {
					return fmt.Errorf("failed to open file in archive: %w", err)
				}

				_, err = io.Copy(outFile, rc)
				if err != nil {
					return fmt.Errorf("failed to copy file: %w", err)
				}

				outFile.Close()
				rc.Close()
			}

			// read content of latest-support-packet/support_packet.yaml
			bSupPack, err := os.ReadFile(filepath.Join(latestSupportPacketFolder, "support_packet.yaml"))
			if err != nil {
				return fmt.Errorf("failed to read support_packet.yaml: %w", err)
			}
			// unmarshal the content in sp
			err = yaml.Unmarshal(bSupPack, &sp)
			if err != nil {
				return fmt.Errorf("failed to unmarshal support_packet.yaml: %w", err)
			}
		}

		// in cs-repo/docker-compose.yml, replace the mattermost image version with the server_version
		bDockerCompose, err := os.ReadFile(filepath.Join(csReproDest, "docker-compose.yml"))
		if err != nil {
			return fmt.Errorf("failed to read docker-compose.yml: %w", err)
		}
		dockerComposeMap := map[string]any{}
		err = yaml.Unmarshal(bDockerCompose, &dockerComposeMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal docker-compose.yml: %w", err)
		}
		dockerComposeMap["name"] = "cs-repro-" + ticketNumberStr
		if sp.ServerVersion != "" {
			dockerComposeMap["services"].(map[string]any)["mattermost"].(map[string]any)["image"] = fmt.Sprintf("mattermost/mattermost-enterprise-edition:%s", sp.ServerVersion)
		}
		bDockerCompose, err = yaml.Marshal(dockerComposeMap)
		if err != nil {
			return fmt.Errorf("failed to marshal docker-compose.yml: %w", err)
		}
		err = os.WriteFile(filepath.Join(csReproDest, "docker-compose.yml"), bDockerCompose, 0644)
		if err != nil {
			return fmt.Errorf("failed to write docker-compose.yml: %w", err)
		}

		log.Println("Everything is ready")

		return nil
	},
}

func replaceInFolder(rootPath, oldStr, newStr string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return replaceInFile(path, oldStr, newStr)
		}

		return nil
	})
}

func replaceInFile(filename, oldStr, newStr string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	newContent := strings.ReplaceAll(string(content), oldStr, newStr)

	if newContent != string(content) {
		log.Printf("Replacing in %s", filename)
	}

	err = os.WriteFile(filename, []byte(newContent), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().Bool("get.all-attachments", false, "retrieve all attachments")
	viper.BindPFlag("get.all-attachments", getCmd.Flags().Lookup("get.all-attachments"))

	getCmd.Flags().String("get.cs-repro-repo", "https://github.com/coltoneshaw/CS-Repro-Mattermost", "CS-Repro-Mattermost repository")
	viper.BindPFlag("get.cs-repro-repo", getCmd.Flags().Lookup("get.cs-repro-repo"))
}
