package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)


var ListIterationsCmd = &cobra.Command{
    Use:   "iterations",
    Short: "List iterations",
    Run: func(cmd *cobra.Command, args []string) {
        ab := NewAzureBoards(
            viper.GetString("organization"),
            viper.GetString("project"),
            viper.GetString("pat"),
        )

        item, err := ab.ListIterations()
        if err != nil {
            fmt.Printf("Error listing iterations", err)
            return
        }

        fmt.Printf("Created work item #%d: %s\n", item.ID, item.Fields.Title)
    },
}


func (ab *AzureBoards) ListIterations() (, error) {

    // endpoint = GET https://dev.azure.com/{organization}/{project}/{team}/_apis/work/teamsettings/iterations?api-version=%s
    // work/teamsettings/iterations?api-version=7.1
    endpoint := fmt.Sprintf("work/teamsettings/iterations?api-version=%s", API_VERSION)
	req, err := ab.createRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ab.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()


	return &result, nil
}

