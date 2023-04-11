package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"os"
)

// aboutCmd represents the about command
var aboutCmd = &cobra.Command{
	Use:   "about",
	Short: "About user",
	RunE: func(cmd *cobra.Command, args []string) error {
		accessToken, err := cli.GetConfig()
		if err != nil {
			return fmt.Errorf("error relate to get config file : %v", err)
		}

		var dataInFile cli.DataStoredInFile
		errJm := json.Unmarshal(accessToken, &dataInFile)
		if errJm != nil {
			return errJm
		}

		errFunc, bodyResponse := cli.RequestAbout(dataInFile.AccessToken)
		if errFunc != nil {
			return errFunc
		}

		response := cli.ResponseAbout{}
		errJson := json.Unmarshal(bodyResponse, &response)
		if errJson != nil {
			return errJson
		}

		typeOutput, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		if typeOutput == "json" {
			fmt.Println(string(bodyResponse))
		} else {
			tableAbout := table.NewWriter()
			tableAbout.SetOutputMirror(os.Stdout)
			tableAbout.AppendHeader(table.Row{"", "email", "email_verified", "sub"})
			tableAbout.AppendRows([]table.Row{{"", response.Email, response.EmailVerified, response.Sub}})
			tableAbout.AppendSeparator()
			tableAbout.Render()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(aboutCmd)
	aboutCmd.PersistentFlags().String("output", "", "can use this flag for specify the output type .")
}
