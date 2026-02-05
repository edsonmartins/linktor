package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var flowCmd = &cobra.Command{
	Use:   "flow",
	Short: "Manage flows",
	Long: `Create, view, and manage automation flows.

Examples:
  msgfy flow list
  msgfy flow show fl_abc123
  msgfy flow execute fl_abc123 --conversation cv_xyz
  msgfy flow validate fl_abc123
  msgfy flow export fl_abc123 > flow.json
  msgfy flow import flow.json`,
}

var (
	flowStatus       string
	flowLimit        int
	flowConversation string
	flowName         string
)

var flowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List flows",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if flowStatus != "" {
			params["status"] = flowStatus
		}
		if flowLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", flowLimit)
		}

		resp, err := c.ListFlows(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No flows found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "Status", "Executions"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, flow := range resp.Data {
			table.Append([]string{
				flow.ID,
				flow.Name,
				flow.Status,
				fmt.Sprintf("%d", flow.ExecutionCount),
			})
		}

		table.Render()
		return nil
	},
}

var flowShowCmd = &cobra.Command{
	Use:   "show <flow-id>",
	Short: "Show flow details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		flow, err := c.GetFlow(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(flow, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:          %s\n", flow.ID)
		fmt.Printf("Name:        %s\n", flow.Name)
		fmt.Printf("Description: %s\n", flow.Description)
		fmt.Printf("Status:      %s\n", flow.Status)
		fmt.Printf("Nodes:       %d\n", len(flow.Nodes))
		fmt.Printf("Executions:  %d\n", flow.ExecutionCount)
		fmt.Printf("Created:     %s\n", flow.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated:     %s\n", flow.UpdatedAt.Format("2006-01-02 15:04"))

		if len(flow.Triggers) > 0 {
			fmt.Println("\nTriggers:")
			for _, t := range flow.Triggers {
				fmt.Printf("  - %s\n", t)
			}
		}

		return nil
	},
}

var flowExecuteCmd = &cobra.Command{
	Use:   "execute <flow-id>",
	Short: "Execute a flow manually",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		input := make(map[string]interface{})
		if flowConversation != "" {
			input["conversationId"] = flowConversation
		}

		result, err := c.ExecuteFlow(args[0], input)
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		success("Flow executed")
		fmt.Printf("Execution ID: %s\n", result.ExecutionID)
		fmt.Printf("Status: %s\n", result.Status)

		return nil
	},
}

var flowValidateCmd = &cobra.Command{
	Use:   "validate <flow-id>",
	Short: "Validate a flow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		result, err := c.ValidateFlow(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if result.Valid {
			success("Flow is valid")
		} else {
			errorMsg("Flow has errors")
		}

		if len(result.Errors) > 0 {
			fmt.Println("\nErrors:")
			for _, e := range result.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range result.Warnings {
				fmt.Printf("  - %s\n", w)
			}
		}

		return nil
	},
}

var flowPublishCmd = &cobra.Command{
	Use:   "publish <flow-id>",
	Short: "Publish a flow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		flow, err := c.PublishFlow(args[0])
		if err != nil {
			return err
		}

		success("Flow published: %s", flow.Status)
		return nil
	},
}

var flowUnpublishCmd = &cobra.Command{
	Use:   "unpublish <flow-id>",
	Short: "Unpublish a flow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		flow, err := c.UnpublishFlow(args[0])
		if err != nil {
			return err
		}

		success("Flow unpublished: %s", flow.Status)
		return nil
	},
}

var flowExportCmd = &cobra.Command{
	Use:   "export <flow-id>",
	Short: "Export a flow to JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		flow, err := c.GetFlow(args[0])
		if err != nil {
			return err
		}

		// Create export object without internal fields
		export := map[string]interface{}{
			"name":        flow.Name,
			"description": flow.Description,
			"nodes":       flow.Nodes,
			"triggers":    flow.Triggers,
		}

		data, _ := json.MarshalIndent(export, "", "  ")
		fmt.Println(string(data))

		return nil
	},
}

var flowImportCmd = &cobra.Command{
	Use:   "import <file.json>",
	Short: "Import a flow from JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		var flowData map[string]interface{}
		if err := json.Unmarshal(data, &flowData); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}

		// Override name if provided
		if flowName != "" {
			flowData["name"] = flowName
		}

		flow, err := c.CreateFlow(flowData)
		if err != nil {
			return err
		}

		success("Flow imported: %s", flow.ID)
		return nil
	},
}

var flowDeleteCmd = &cobra.Command{
	Use:   "delete <flow-id>",
	Short: "Delete a flow",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to delete the flow")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteFlow(args[0]); err != nil {
			return err
		}

		success("Flow deleted")
		return nil
	},
}

func init() {
	flowCmd.AddCommand(flowListCmd)
	flowCmd.AddCommand(flowShowCmd)
	flowCmd.AddCommand(flowExecuteCmd)
	flowCmd.AddCommand(flowValidateCmd)
	flowCmd.AddCommand(flowPublishCmd)
	flowCmd.AddCommand(flowUnpublishCmd)
	flowCmd.AddCommand(flowExportCmd)
	flowCmd.AddCommand(flowImportCmd)
	flowCmd.AddCommand(flowDeleteCmd)

	// List flags
	flowListCmd.Flags().StringVar(&flowStatus, "status", "", "Filter by status (draft, published)")
	flowListCmd.Flags().IntVar(&flowLimit, "limit", 20, "Limit results")

	// Execute flags
	flowExecuteCmd.Flags().StringVar(&flowConversation, "conversation", "", "Conversation ID to execute flow on")

	// Import flags
	flowImportCmd.Flags().StringVar(&flowName, "name", "", "Override flow name")

	// Delete flags
	flowDeleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion")
}
