package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var kbCmd = &cobra.Command{
	Use:     "kb",
	Aliases: []string{"knowledge"},
	Short:   "Manage knowledge bases",
	Long: `Create, view, and manage knowledge bases for AI agents.

Examples:
  msgfy kb list
  msgfy kb create --name "Product Docs"
  msgfy kb doc add kb_abc123 --file manual.pdf --title "User Manual"
  msgfy kb doc list kb_abc123
  msgfy kb query kb_abc123 "How do I reset my password?"`,
}

var kbDocCmd = &cobra.Command{
	Use:   "doc",
	Short: "Manage knowledge base documents",
}

var (
	kbName      string
	kbLimit     int
	docTitle    string
	docFile     string
	docURL      string
	queryLimit  int
	queryMinScore float64
)

var kbListCmd = &cobra.Command{
	Use:   "list",
	Short: "List knowledge bases",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if kbLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", kbLimit)
		}

		resp, err := c.ListKnowledgeBases(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No knowledge bases found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "Documents", "Chunks", "Status"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, kb := range resp.Data {
			table.Append([]string{
				kb.ID,
				kb.Name,
				fmt.Sprintf("%d", kb.DocumentCount),
				fmt.Sprintf("%d", kb.ChunkCount),
				kb.Status,
			})
		}

		table.Render()
		return nil
	},
}

var kbShowCmd = &cobra.Command{
	Use:   "show <kb-id>",
	Short: "Show knowledge base details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		kb, err := c.GetKnowledgeBase(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(kb, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:         %s\n", kb.ID)
		fmt.Printf("Name:       %s\n", kb.Name)
		fmt.Printf("Status:     %s\n", kb.Status)
		fmt.Printf("Documents:  %d\n", kb.DocumentCount)
		fmt.Printf("Chunks:     %d\n", kb.ChunkCount)
		fmt.Printf("Created:    %s\n", kb.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated:    %s\n", kb.UpdatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

var kbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new knowledge base",
	RunE: func(cmd *cobra.Command, args []string) error {
		if kbName == "" {
			return fmt.Errorf("--name is required")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		input := map[string]interface{}{
			"name": kbName,
		}

		kb, err := c.CreateKnowledgeBase(input)
		if err != nil {
			return err
		}

		success("Knowledge base created: %s", kb.ID)
		info("Add documents with: msgfy kb doc add %s --file <file>", kb.ID)

		return nil
	},
}

var kbDeleteCmd = &cobra.Command{
	Use:   "delete <kb-id>",
	Short: "Delete a knowledge base",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to delete the knowledge base")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteKnowledgeBase(args[0]); err != nil {
			return err
		}

		success("Knowledge base deleted")
		return nil
	},
}

var kbQueryCmd = &cobra.Command{
	Use:   "query <kb-id> <question>",
	Short: "Query a knowledge base",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		kbID := args[0]
		question := args[1]
		for i := 2; i < len(args); i++ {
			question += " " + args[i]
		}

		input := map[string]interface{}{
			"query": question,
			"limit": queryLimit,
		}
		if queryMinScore > 0 {
			input["minScore"] = queryMinScore
		}

		results, err := c.QueryKnowledgeBase(kbID, input)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			info("No results found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Results (score > %.2f):\n\n", queryMinScore)
		for i, r := range results {
			fmt.Printf("%d. [%.2f] %s\n", i+1, r.Score, r.Title)

			// Truncate content for display
			content := r.Content
			if len(content) > 200 {
				content = content[:197] + "..."
			}
			fmt.Printf("   %s\n\n", content)
		}

		return nil
	},
}

// Document subcommands

var kbDocListCmd = &cobra.Command{
	Use:   "list <kb-id>",
	Short: "List documents in a knowledge base",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if kbLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", kbLimit)
		}

		resp, err := c.ListDocuments(args[0], params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No documents found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Title", "Type", "Chunks", "Status"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, doc := range resp.Data {
			table.Append([]string{
				doc.ID,
				doc.Title,
				doc.Type,
				fmt.Sprintf("%d", doc.ChunkCount),
				doc.Status,
			})
		}

		table.Render()
		return nil
	},
}

var kbDocAddCmd = &cobra.Command{
	Use:   "add <kb-id>",
	Short: "Add a document to a knowledge base",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if docFile == "" && docURL == "" {
			return fmt.Errorf("--file or --url is required")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		kbID := args[0]

		input := make(map[string]interface{})
		if docTitle != "" {
			input["title"] = docTitle
		}

		var doc *client.Document

		if docFile != "" {
			// Upload file
			doc, err = c.UploadDocument(kbID, docFile, input)
		} else {
			// Add from URL
			input["url"] = docURL
			doc, err = c.AddDocument(kbID, input)
		}

		if err != nil {
			return err
		}

		success("Document added: %s", doc.ID)
		info("Processing... Status: %s", doc.Status)

		return nil
	},
}

var kbDocShowCmd = &cobra.Command{
	Use:   "show <kb-id> <doc-id>",
	Short: "Show document details",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		doc, err := c.GetDocument(args[0], args[1])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(doc, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:       %s\n", doc.ID)
		fmt.Printf("Title:    %s\n", doc.Title)
		fmt.Printf("Type:     %s\n", doc.Type)
		fmt.Printf("Status:   %s\n", doc.Status)
		fmt.Printf("Chunks:   %d\n", doc.ChunkCount)
		fmt.Printf("Created:  %s\n", doc.CreatedAt.Format("2006-01-02 15:04"))

		return nil
	},
}

var kbDocDeleteCmd = &cobra.Command{
	Use:   "delete <kb-id> <doc-id>",
	Short: "Delete a document",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to delete the document")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteDocument(args[0], args[1]); err != nil {
			return err
		}

		success("Document deleted")
		return nil
	},
}

var kbDocReprocessCmd = &cobra.Command{
	Use:   "reprocess <kb-id> <doc-id>",
	Short: "Reprocess a document",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		doc, err := c.ReprocessDocument(args[0], args[1])
		if err != nil {
			return err
		}

		success("Document queued for reprocessing: %s", doc.Status)
		return nil
	},
}

func init() {
	kbCmd.AddCommand(kbListCmd)
	kbCmd.AddCommand(kbShowCmd)
	kbCmd.AddCommand(kbCreateCmd)
	kbCmd.AddCommand(kbDeleteCmd)
	kbCmd.AddCommand(kbQueryCmd)
	kbCmd.AddCommand(kbDocCmd)

	// Doc subcommands
	kbDocCmd.AddCommand(kbDocListCmd)
	kbDocCmd.AddCommand(kbDocAddCmd)
	kbDocCmd.AddCommand(kbDocShowCmd)
	kbDocCmd.AddCommand(kbDocDeleteCmd)
	kbDocCmd.AddCommand(kbDocReprocessCmd)

	// List flags
	kbListCmd.Flags().IntVar(&kbLimit, "limit", 20, "Limit results")

	// Create flags
	kbCreateCmd.Flags().StringVar(&kbName, "name", "", "Knowledge base name (required)")

	// Delete flags
	kbDeleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion")

	// Query flags
	kbQueryCmd.Flags().IntVar(&queryLimit, "limit", 5, "Number of results")
	kbQueryCmd.Flags().Float64Var(&queryMinScore, "min-score", 0.7, "Minimum relevance score")

	// Doc list flags
	kbDocListCmd.Flags().IntVar(&kbLimit, "limit", 20, "Limit results")

	// Doc add flags
	kbDocAddCmd.Flags().StringVar(&docFile, "file", "", "File path to upload")
	kbDocAddCmd.Flags().StringVar(&docURL, "url", "", "URL to fetch document from")
	kbDocAddCmd.Flags().StringVar(&docTitle, "title", "", "Document title")

	// Doc delete flags
	kbDocDeleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion")
}
