package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.com/keibiengine/keibi-engine/pkg/cli"
	"os"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Logging in into kaytu",
	Long:  `Logging into kaytu using device authentication mechanism`,
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceCode, err := cli.RequestDeviceCode()
		if err != nil {
			return err
		}

		AT, errAccessToken := cli.AccessToken(deviceCode)
		if errAccessToken != nil {
			return errAccessToken
		}

		//save accessToken to the file :
		var data cli.DataStoredInFile
		data.AccessToken = AT
		accessToken, errJm := json.Marshal(data)
		if errJm != nil {
			return errJm
		}
		home := os.Getenv("HOME")
		if _, errStat := os.Stat(home + "/.kaytu/config.json"); errStat != nil {

			file, errFil := os.Create(home + "/.kaytu/config.json")
			if errFil != nil {
				return errFil
			}

			_, errWrite := file.WriteString(string(accessToken))
			if errWrite != nil {
				fmt.Println("error belong to writing accessToken into file : ")
				return errWrite
			}
		} else {
			errRemove := os.Remove(home + "/.kaytu/config.json")
			if errRemove != nil {
				fmt.Println("error relate to removing file accessToken: ")
				return errRemove
			}

			file, errFil := os.Create(home + "/.kaytu/config.json")
			if errFil != nil {
				return errFil
			}

			_, errWrite := file.WriteString(string(accessToken))
			if errWrite != nil {
				fmt.Println("error belong to writing accessToken into file : ")
				return errWrite
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
