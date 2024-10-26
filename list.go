package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ListCmd = &cobra.Command{
    Use:   "list",
    Short: "List work items",
    Run: func(cmd *cobra.Command, args []string) {
        ab := NewAzureBoards(
            viper.GetString("organization"),
            viper.GetString("project"),
            viper.GetString("pat"),
        )

        items, err := ab.GetWorkItems()
        if err != nil {
            fmt.Printf("Error getting work items: %s\n", err)
            return
        }

        if len(items) == 0 {
            fmt.Println("No work items found")
            return
        }

        fmt.Println("Work Items:")
        for _, item := range items {
            fmt.Printf("#%d: %s [%s]\n", item.ID, item.Fields.Title, item.Fields.State)
        }
    },
}

var GetCmd = &cobra.Command{
    Use:   "get",
    Short: "Get work item by ID",
    Run: func(cmd *cobra.Command, args []string) {
        ab := NewAzureBoards(
            viper.GetString("organization"),
            viper.GetString("project"),
            viper.GetString("pat"),
        )

        items, err := ab.GetWorkItemsList(args)
        if err != nil {
            fmt.Printf("Error getting work items: %s\n", err)
            return
        }

        if len(items) == 0 {
            fmt.Println("No work items found")
            return
        }

        fmt.Println("Work Items:")
        for _, item := range items {
            fmt.Printf("#%d: %s [%s]\n", item.ID, item.Fields.Title, item.Fields.State)
        }
    },
}

func (ab *AzureBoards) GetWorkItems() ([]WorkItem, error) {
	// First, get work item IDs using WIQL
	query := map[string]string{
		"query": "SELECT [System.Id], [System.Title], [System.State] FROM WorkItems WHERE [System.TeamProject] = @project ORDER BY [System.ChangedDate] DESC",
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("error marshaling query: %v", err)
	}

	req, err := ab.createRequest("POST", fmt.Sprintf("wit/wiql?api-version=%s", API_VERSION), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating WIQL request: %v", err)
	}

	resp, err := ab.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing WIQL request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WIQL request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var wiqlResp WiqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&wiqlResp); err != nil {
		return nil, fmt.Errorf("error decoding WIQL response: %v", err)
	}

	if len(wiqlResp.WorkItems) == 0 {
		return []WorkItem{}, nil
	}

	// Build the IDs string for batch request
	var ids string
	for i, item := range wiqlResp.WorkItems {
		if i > 0 {
			ids += ","
		}
		ids += fmt.Sprintf("%d", item.ID)
	}

    return ab.GetWorkItemDetails(ids)
}

func (ab *AzureBoards) GetWorkItemsList(idList []string) ([]WorkItem, error) {
	// Build the IDs string for batch request
	var ids string
	for i, item := range idList {
		if i > 0 {
			ids += ","
		}
		ids += fmt.Sprintf("%s", item)
	}
    return ab.GetWorkItemDetails(ids);
}

