package cli

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
)

type OutputType string

func PrintOutputForWorkspaces(obj interface{}, typeOutput string) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("[printoutput] : %v", err)
	}

	if typeOutput == "json" {
		fmt.Println(string(bytes))
		return nil
	}
	
	var fields []map[string]interface{}
	err = json.Unmarshal(bytes, &fields)
	if err != nil {
		return fmt.Errorf("[printoutput] : %v", err)
	}
	printTable := table.NewWriter()
	printTable.SetOutputMirror(os.Stdout)

	var headers []interface{}
	var record []interface{}
	for _, vl := range fields {
		for key, value := range vl {
			headers = append(headers, key)
			record = append(record, value)
		}
	}
	printTable.AppendHeader(headers)
	printTable.AppendRows([]table.Row{record})
	printTable.AppendSeparator()
	printTable.Render()
	return nil
}
func PrintOutputForAbout(obj interface{}, typeOutput string) error {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("[printoutput] : %v", err)
	}

	if typeOutput == "json" {
		fmt.Println(string(bytes))
		return nil
	}

	var fields map[string]interface{}
	err = json.Unmarshal(bytes, &fields)
	if err != nil {
		return fmt.Errorf("[printoutput] : %v", err)
	}
	printTable := table.NewWriter()
	printTable.SetOutputMirror(os.Stdout)

	var headers []interface{}
	var record []interface{}
	for key, value := range fields {
		headers = append(headers, key)
		record = append(record, value)
	}
	printTable.AppendHeader(headers)
	printTable.AppendRows([]table.Row{record})
	printTable.AppendSeparator()
	printTable.Render()
	return nil
}
