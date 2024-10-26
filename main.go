package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type AzureBoards struct {
	organization string
	project     string
	pat         string
	baseURL     string
	client      *http.Client
}

// WiqlResponse represents the response from a WIQL query
type WiqlResponse struct {
	WorkItems []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	} `json:"workItems"`
}

// WorkItemFields represents the fields in a work item
type WorkItemFields struct {
	Title       string `json:"System.Title"`
	State       string `json:"System.State"`
	Description string `json:"System.Description"`
}

// WorkItem represents a single work item with its fields
type WorkItem struct {
	ID     int            `json:"id"`
	Fields WorkItemFields `json:"fields"`
}

// BatchWorkItemRequest represents the request for getting multiple work items
type BatchWorkItemResponse struct {
	Value []WorkItem `json:"value"`
}

func NewAzureBoards(org, project, pat string) *AzureBoards {
	return &AzureBoards{
		organization: org,
		project:     project,
		pat:         pat,
		baseURL:     fmt.Sprintf("https://dev.azure.com/%s/%s/_apis", org, project),
		client:      &http.Client{},
	}
}

func (ab *AzureBoards) createRequest(method, path string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s", ab.baseURL, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(":" + ab.pat))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	return req, nil
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

func (ab *AzureBoards) GetWorkItemDetails(ids string) ([]WorkItem, error) {
	// Get work item details
	batchURL := fmt.Sprintf("wit/workitems?ids=%s&api-version=%s", ids, API_VERSION)
    req, err := ab.createRequest("GET", batchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating batch request: %v", err)
	}

    resp, err := ab.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing batch request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var batchResp BatchWorkItemResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("error decoding batch response: %v", err)
	}

	return batchResp.Value, nil
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

var API_VERSION = "7.1"

func main() {
	var rootCmd = &cobra.Command{
		Use:   "azboards",
		Short: "Azure Boards CLI tool",
		Long:  `A command line interface for managing Azure Boards work items`,
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.azboards")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Error reading config file: %s\n", err)
			os.Exit(1)
		}
	}

	var listCmd = &cobra.Command{
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

	var getCmd = &cobra.Command{
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

	var createCmd = &cobra.Command{
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

	createCmd.Flags().String("title", "", "Title of the work item")
	createCmd.Flags().String("description", "", "Description of the work item")
	createCmd.Flags().String("type", "Task", "Type of work item (Task, Bug, etc.)")
	createCmd.MarkFlagRequired("title")

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(getCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
