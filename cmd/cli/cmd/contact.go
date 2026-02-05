package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"

	"github.com/linktor/msgfy/internal/client"
)

var contactCmd = &cobra.Command{
	Use:     "contact",
	Aliases: []string{"contacts"},
	Short:   "Manage contacts",
	Long: `Create, view, and manage contacts.

Examples:
  msgfy contact list --search "João"
  msgfy contact show ct_abc123
  msgfy contact create --name "Maria" --phone "+5544999999999"
  msgfy contact import contacts.csv
  msgfy contact export --format csv`,
}

var (
	contactSearch  string
	contactTags    string
	contactLimit   int
	contactName    string
	contactPhone   string
	contactEmail   string
	contactSetVars []string
	importMapping  string
	exportFormat   string
	keepContact    string
)

var contactListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{}
		if contactSearch != "" {
			params["search"] = contactSearch
		}
		if contactTags != "" {
			params["tags"] = contactTags
		}
		if contactLimit > 0 {
			params["limit"] = fmt.Sprintf("%d", contactLimit)
		}

		resp, err := c.ListContacts(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No contacts found")
			return nil
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if outputFormat == "ids" {
			for _, ct := range resp.Data {
				fmt.Println(ct.ID)
			}
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Name", "Phone", "Email", "Tags"})
		table.SetBorder(false)
		table.SetColumnSeparator("  ")

		for _, ct := range resp.Data {
			tags := strings.Join(ct.Tags, ", ")
			if len(tags) > 20 {
				tags = tags[:17] + "..."
			}
			table.Append([]string{
				ct.ID,
				ct.Name,
				ct.Phone,
				ct.Email,
				tags,
			})
		}

		table.Render()
		return nil
	},
}

var contactShowCmd = &cobra.Command{
	Use:   "show <contact-id>",
	Short: "Show contact details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		ct, err := c.GetContact(args[0])
		if err != nil {
			return err
		}

		if outputFormat == "json" {
			data, _ := json.MarshalIndent(ct, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID:      %s\n", ct.ID)
		fmt.Printf("Name:    %s\n", ct.Name)
		fmt.Printf("Phone:   %s\n", ct.Phone)
		fmt.Printf("Email:   %s\n", ct.Email)
		fmt.Printf("Tags:    %s\n", strings.Join(ct.Tags, ", "))
		fmt.Printf("Created: %s\n", ct.CreatedAt.Format("2006-01-02 15:04"))

		if len(ct.CustomFields) > 0 {
			fmt.Println("\nCustom Fields:")
			for k, v := range ct.CustomFields {
				fmt.Printf("  %s: %v\n", k, v)
			}
		}

		return nil
	},
}

var contactCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		if contactName == "" {
			return fmt.Errorf("--name is required")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		input := map[string]interface{}{
			"name": contactName,
		}

		if contactPhone != "" {
			input["phone"] = contactPhone
		}
		if contactEmail != "" {
			input["email"] = contactEmail
		}
		if contactTags != "" {
			input["tags"] = strings.Split(contactTags, ",")
		}

		// Parse custom fields from --set flags
		if len(contactSetVars) > 0 {
			customFields := make(map[string]interface{})
			for _, s := range contactSetVars {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) == 2 {
					customFields[parts[0]] = parts[1]
				}
			}
			if len(customFields) > 0 {
				input["customFields"] = customFields
			}
		}

		ct, err := c.CreateContact(input)
		if err != nil {
			return err
		}

		success("Contact created: %s", ct.ID)
		return nil
	},
}

var contactUpdateCmd = &cobra.Command{
	Use:   "update <contact-id>",
	Short: "Update a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		input := make(map[string]interface{})

		if contactName != "" {
			input["name"] = contactName
		}
		if contactPhone != "" {
			input["phone"] = contactPhone
		}
		if contactEmail != "" {
			input["email"] = contactEmail
		}
		if contactTags != "" {
			input["tags"] = strings.Split(contactTags, ",")
		}

		// Parse custom fields from --set flags
		if len(contactSetVars) > 0 {
			customFields := make(map[string]interface{})
			for _, s := range contactSetVars {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) == 2 {
					customFields[parts[0]] = parts[1]
				}
			}
			if len(customFields) > 0 {
				input["customFields"] = customFields
			}
		}

		if len(input) == 0 {
			return fmt.Errorf("no updates specified")
		}

		_, err = c.UpdateContact(args[0], input)
		if err != nil {
			return err
		}

		success("Contact updated")
		return nil
	},
}

var contactDeleteCmd = &cobra.Command{
	Use:   "delete <contact-id>",
	Short: "Delete a contact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !confirmDelete {
			return fmt.Errorf("use --confirm to delete the contact")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteContact(args[0]); err != nil {
			return err
		}

		success("Contact deleted")
		return nil
	},
}

var contactImportCmd = &cobra.Command{
	Use:   "import <file.csv>",
	Short: "Import contacts from CSV",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		file, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		reader := csv.NewReader(file)

		// Read header
		header, err := reader.Read()
		if err != nil {
			return fmt.Errorf("failed to read CSV header: %w", err)
		}

		// Parse column mapping
		mapping := parseMapping(importMapping, header)

		// Read records
		records, err := reader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read CSV: %w", err)
		}

		imported := 0
		failed := 0
		var errors []string

		for i, record := range records {
			input := make(map[string]interface{})

			for j, col := range header {
				if j >= len(record) {
					continue
				}
				value := strings.TrimSpace(record[j])
				if value == "" {
					continue
				}

				// Map column to field
				field := mapping[col]
				if field == "" {
					field = strings.ToLower(col)
				}

				switch field {
				case "name":
					input["name"] = value
				case "phone":
					input["phone"] = value
				case "email":
					input["email"] = value
				case "tags":
					input["tags"] = strings.Split(value, ",")
				default:
					// Custom field
					if input["customFields"] == nil {
						input["customFields"] = make(map[string]interface{})
					}
					input["customFields"].(map[string]interface{})[field] = value
				}
			}

			if input["name"] == nil {
				errors = append(errors, fmt.Sprintf("Row %d: missing name", i+2))
				failed++
				continue
			}

			_, err := c.CreateContact(input)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Row %d: %v", i+2, err))
				failed++
				continue
			}

			imported++
		}

		if imported > 0 {
			success("%d contacts imported", imported)
		}
		if failed > 0 {
			errorMsg("%d failed", failed)
			for _, e := range errors {
				fmt.Printf("  %s\n", e)
			}
		}

		return nil
	},
}

var contactExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		params := map[string]string{
			"limit": "1000",
		}
		if contactTags != "" {
			params["tags"] = contactTags
		}

		resp, err := c.ListContacts(params)
		if err != nil {
			return err
		}

		if len(resp.Data) == 0 {
			info("No contacts to export")
			return nil
		}

		switch exportFormat {
		case "json":
			data, _ := json.MarshalIndent(resp.Data, "", "  ")
			fmt.Println(string(data))

		case "csv":
			writer := csv.NewWriter(os.Stdout)
			defer writer.Flush()

			// Header
			writer.Write([]string{"id", "name", "phone", "email", "tags"})

			for _, ct := range resp.Data {
				writer.Write([]string{
					ct.ID,
					ct.Name,
					ct.Phone,
					ct.Email,
					strings.Join(ct.Tags, ","),
				})
			}

		default:
			return fmt.Errorf("unsupported format: %s (use json or csv)", exportFormat)
		}

		return nil
	},
}

var contactMergeCmd = &cobra.Command{
	Use:   "merge <contact-id> <contact-id>",
	Short: "Merge two contacts",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if keepContact == "" {
			return fmt.Errorf("--keep is required to specify which contact to keep")
		}

		if keepContact != args[0] && keepContact != args[1] {
			return fmt.Errorf("--keep must be one of the contact IDs")
		}

		c, err := client.New()
		if err != nil {
			return err
		}

		// Determine source and target
		var sourceID, targetID string
		if keepContact == args[0] {
			targetID = args[0]
			sourceID = args[1]
		} else {
			targetID = args[1]
			sourceID = args[0]
		}

		// Get both contacts
		source, err := c.GetContact(sourceID)
		if err != nil {
			return fmt.Errorf("failed to get source contact: %w", err)
		}

		target, err := c.GetContact(targetID)
		if err != nil {
			return fmt.Errorf("failed to get target contact: %w", err)
		}

		// Merge data (fill in missing fields)
		updates := make(map[string]interface{})

		if target.Phone == "" && source.Phone != "" {
			updates["phone"] = source.Phone
		}
		if target.Email == "" && source.Email != "" {
			updates["email"] = source.Email
		}

		// Merge tags
		tagSet := make(map[string]bool)
		for _, t := range target.Tags {
			tagSet[t] = true
		}
		for _, t := range source.Tags {
			tagSet[t] = true
		}
		var mergedTags []string
		for t := range tagSet {
			mergedTags = append(mergedTags, t)
		}
		if len(mergedTags) > len(target.Tags) {
			updates["tags"] = mergedTags
		}

		// Merge custom fields
		if len(source.CustomFields) > 0 {
			mergedFields := make(map[string]interface{})
			for k, v := range target.CustomFields {
				mergedFields[k] = v
			}
			for k, v := range source.CustomFields {
				if _, exists := mergedFields[k]; !exists {
					mergedFields[k] = v
				}
			}
			if len(mergedFields) > len(target.CustomFields) {
				updates["customFields"] = mergedFields
			}
		}

		// Update target if there are changes
		if len(updates) > 0 {
			_, err = c.UpdateContact(targetID, updates)
			if err != nil {
				return fmt.Errorf("failed to update target contact: %w", err)
			}
		}

		// Delete source contact
		if err := c.DeleteContact(sourceID); err != nil {
			return fmt.Errorf("failed to delete source contact: %w", err)
		}

		success("Contacts merged. Kept: %s", targetID)
		return nil
	},
}

func init() {
	contactCmd.AddCommand(contactListCmd)
	contactCmd.AddCommand(contactShowCmd)
	contactCmd.AddCommand(contactCreateCmd)
	contactCmd.AddCommand(contactUpdateCmd)
	contactCmd.AddCommand(contactDeleteCmd)
	contactCmd.AddCommand(contactImportCmd)
	contactCmd.AddCommand(contactExportCmd)
	contactCmd.AddCommand(contactMergeCmd)

	// List flags
	contactListCmd.Flags().StringVar(&contactSearch, "search", "", "Search contacts by name/phone/email")
	contactListCmd.Flags().StringVar(&contactTags, "tags", "", "Filter by tags (comma-separated)")
	contactListCmd.Flags().IntVar(&contactLimit, "limit", 20, "Limit results")

	// Create flags
	contactCreateCmd.Flags().StringVar(&contactName, "name", "", "Contact name (required)")
	contactCreateCmd.Flags().StringVar(&contactPhone, "phone", "", "Phone number")
	contactCreateCmd.Flags().StringVar(&contactEmail, "email", "", "Email address")
	contactCreateCmd.Flags().StringVar(&contactTags, "tags", "", "Tags (comma-separated)")
	contactCreateCmd.Flags().StringArrayVar(&contactSetVars, "set", nil, "Set custom field (key=value)")

	// Update flags
	contactUpdateCmd.Flags().StringVar(&contactName, "name", "", "Contact name")
	contactUpdateCmd.Flags().StringVar(&contactPhone, "phone", "", "Phone number")
	contactUpdateCmd.Flags().StringVar(&contactEmail, "email", "", "Email address")
	contactUpdateCmd.Flags().StringVar(&contactTags, "tags", "", "Tags (comma-separated)")
	contactUpdateCmd.Flags().StringArrayVar(&contactSetVars, "set", nil, "Set custom field (key=value)")

	// Delete flags
	contactDeleteCmd.Flags().BoolVar(&confirmDelete, "confirm", false, "Confirm deletion")

	// Import flags
	contactImportCmd.Flags().StringVar(&importMapping, "mapping", "", "Column mapping (field:Column,field:Column)")

	// Export flags
	contactExportCmd.Flags().StringVar(&exportFormat, "format", "csv", "Export format (csv, json)")
	contactExportCmd.Flags().StringVar(&contactTags, "tags", "", "Filter by tags")

	// Merge flags
	contactMergeCmd.Flags().StringVar(&keepContact, "keep", "", "Contact ID to keep (required)")
}

func parseMapping(mappingStr string, header []string) map[string]string {
	mapping := make(map[string]string)

	if mappingStr == "" {
		return mapping
	}

	parts := strings.Split(mappingStr, ",")
	for _, p := range parts {
		kv := strings.SplitN(p, ":", 2)
		if len(kv) == 2 {
			// field:Column -> Column -> field
			mapping[kv[1]] = kv[0]
		}
	}

	return mapping
}

// readLines reads non-empty lines from a file
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.Scanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	return lines, scanner.Err()
}
