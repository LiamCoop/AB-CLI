package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var API_VERSION = "7.1"

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

	CreateCmd.Flags().String("title", "", "Title of the work item")
	CreateCmd.Flags().String("description", "", "Description of the work item")
	CreateCmd.Flags().String("type", "Task", "Type of work item (Task, Bug, etc.)")
	CreateCmd.MarkFlagRequired("title")

	rootCmd.AddCommand(ListCmd)
	rootCmd.AddCommand(CreateCmd)
	rootCmd.AddCommand(GetCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
