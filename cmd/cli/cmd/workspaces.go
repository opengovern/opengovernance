/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
)

// workspacesCmd represents the workspaces command
var workspacesCmd = &cobra.Command{
	Use:   "workspaces",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		err := request()
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(workspacesCmd)
}

type responseWorkSpaces struct {
	WorkspaceID           string `json:"id"`
	WorkspaceOwnerID      string `json:"ownerId"`
	WorkspaceTier         string `json:"tier"`
	WorkspaceDescription  string `json:"description"`
	WorkspaceUri          string `json:"uri"`
	WorkspaceName         string `json:"name"`
	WorkspaceState        string `json:"status"`
	WorkspaceCreationTime string `json:"createdAt"`
	WorkspaceVersion      string `json:"version"`
}

const urlWorkspace string = "https://app.dev.keibi.io/keibi/workspace/api/v1/workspaces"

func request() error {
	req, err := http.NewRequest("GET", urlWorkspace, nil)
	if err != nil {
		fmt.Println("error related to request in workspaces: ")
		return err
	}
	home := os.Getenv("HOME")
	accessTokenFile, errFile := os.ReadFile(home + "/.kaytu/auth/accessToken.txt")
	if errFile != nil {
		fmt.Println("error relate to reading accessToken file in workspaces: ")
		return errFile
	}
	req.Header.Set("Authorization", "Bearer "+string(accessTokenFile))
	res, errRead := http.DefaultClient.Do(req)
	if errRead != nil {
		fmt.Println("error relate to response in workspaces:")
		return errRead
	}
	body, errBody := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if errBody != nil {
		fmt.Println("error relate to reading response in workspaces:")
		return errBody
	}
	response := responseWorkSpaces{}
	//fmt.Println(string(body))
	errJson := json.Unmarshal(body, &response)
	if errJson != nil {
		fmt.Println("error relate to jsonUnmarshal in workspace: ")
		return errJson
	}
	fmt.Println(response.WorkspaceName)
	fmt.Printf("\n%v", response.WorkspaceID)
	fmt.Printf("\n%v", response.WorkspaceState)
	fmt.Printf("\n%v", response.WorkspaceCreationTime)
	fmt.Printf("\n%v", response.WorkspaceVersion)
	return nil
}
