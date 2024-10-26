package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var CreateCmd = &cobra.Command{
    Use:   "create",
    Short: "Create a new work item",
    Run: func(cmd *cobra.Command, args []string) {
        title, _ := cmd.Flags().GetString("title")
        description, _ := cmd.Flags().GetString("description")
        itemType, _ := cmd.Flags().GetString("type")

        ab := NewAzureBoards(
            viper.GetString("organization"),
            viper.GetString("project"),
            viper.GetString("pat"),
        )

        item, err := ab.CreateWorkItem(title, description, itemType)
        if err != nil {
            fmt.Printf("Error creating work item: %s\n", err)
            return
        }

        fmt.Printf("Created work item #%d: %s\n", item.ID, item.Fields.Title)
    },
}

func (ab *AzureBoards) CreateWorkItem(title, description, itemType string) (*WorkItem, error) {
	operations := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/fields/System.Title",
			"value": title,
		},
		{
			"op":    "add",
			"path":  "/fields/System.Description",
			"value": description,
		},
	}

	body, err := json.Marshal(operations)
	if err != nil {
		return nil, err
	}

	req, err := ab.createRequest("POST", fmt.Sprintf("wit/workitems/$%s?api-version=%s", itemType, API_VERSION), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	resp, err := ab.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result WorkItem
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

