package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	apiURL   string
	token    string
	tenantID string
	output   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ovncp-tenant",
		Short: "OVN Control Platform Tenant Management CLI",
		Long:  `A command-line tool for managing tenants in OVN Control Platform`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", getEnvOrDefault("OVNCP_URL", "http://localhost:8080"), "API URL")
	rootCmd.PersistentFlags().StringVar(&token, "token", os.Getenv("OVNCP_TOKEN"), "API token")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format (table, json, yaml)")

	// Tenant commands
	tenantCmd := &cobra.Command{
		Use:   "tenant",
		Short: "Manage tenants",
	}

	// List tenants
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tenants",
		RunE:  listTenants,
	}
	listCmd.Flags().StringVar(&tenantID, "parent", "", "Filter by parent tenant")

	// Create tenant
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new tenant",
		RunE:  createTenant,
	}
	createCmd.Flags().String("name", "", "Tenant name (required)")
	createCmd.Flags().String("display-name", "", "Display name")
	createCmd.Flags().String("type", "project", "Tenant type (organization, project, environment)")
	createCmd.Flags().String("parent", "", "Parent tenant ID")
	createCmd.Flags().Int("max-switches", 100, "Maximum switches quota")
	createCmd.Flags().Int("max-routers", 50, "Maximum routers quota")
	createCmd.MarkFlagRequired("name")

	// Get tenant
	getCmd := &cobra.Command{
		Use:   "get [tenant-id]",
		Short: "Get tenant details",
		Args:  cobra.ExactArgs(1),
		RunE:  getTenant,
	}

	// Update tenant
	updateCmd := &cobra.Command{
		Use:   "update [tenant-id]",
		Short: "Update a tenant",
		Args:  cobra.ExactArgs(1),
		RunE:  updateTenant,
	}
	updateCmd.Flags().String("display-name", "", "New display name")
	updateCmd.Flags().Int("max-switches", -1, "Update switches quota")
	updateCmd.Flags().Int("max-routers", -1, "Update routers quota")

	// Delete tenant
	deleteCmd := &cobra.Command{
		Use:   "delete [tenant-id]",
		Short: "Delete a tenant",
		Args:  cobra.ExactArgs(1),
		RunE:  deleteTenant,
	}

	// Usage command
	usageCmd := &cobra.Command{
		Use:   "usage [tenant-id]",
		Short: "Show resource usage for a tenant",
		Args:  cobra.ExactArgs(1),
		RunE:  showUsage,
	}

	tenantCmd.AddCommand(listCmd, createCmd, getCmd, updateCmd, deleteCmd, usageCmd)

	// Member commands
	memberCmd := &cobra.Command{
		Use:   "member",
		Short: "Manage tenant members",
	}

	// List members
	listMembersCmd := &cobra.Command{
		Use:   "list [tenant-id]",
		Short: "List tenant members",
		Args:  cobra.ExactArgs(1),
		RunE:  listMembers,
	}

	// Add member
	addMemberCmd := &cobra.Command{
		Use:   "add [tenant-id] [user-id]",
		Short: "Add a member to tenant",
		Args:  cobra.ExactArgs(2),
		RunE:  addMember,
	}
	addMemberCmd.Flags().String("role", "viewer", "Member role (admin, operator, viewer)")

	// Remove member
	removeMemberCmd := &cobra.Command{
		Use:   "remove [tenant-id] [user-id]",
		Short: "Remove a member from tenant",
		Args:  cobra.ExactArgs(2),
		RunE:  removeMember,
	}

	memberCmd.AddCommand(listMembersCmd, addMemberCmd, removeMemberCmd)

	// API key commands
	apiKeyCmd := &cobra.Command{
		Use:   "apikey",
		Short: "Manage API keys",
	}

	// List API keys
	listKeysCmd := &cobra.Command{
		Use:   "list [tenant-id]",
		Short: "List API keys for a tenant",
		Args:  cobra.ExactArgs(1),
		RunE:  listAPIKeys,
	}

	// Create API key
	createKeyCmd := &cobra.Command{
		Use:   "create [tenant-id]",
		Short: "Create an API key",
		Args:  cobra.ExactArgs(1),
		RunE:  createAPIKey,
	}
	createKeyCmd.Flags().String("name", "", "Key name (required)")
	createKeyCmd.Flags().StringSlice("scopes", []string{"read"}, "Key scopes")
	createKeyCmd.Flags().Int("expires-in", 365, "Expiration in days")
	createKeyCmd.MarkFlagRequired("name")

	// Delete API key
	deleteKeyCmd := &cobra.Command{
		Use:   "delete [tenant-id] [key-id]",
		Short: "Delete an API key",
		Args:  cobra.ExactArgs(2),
		RunE:  deleteAPIKey,
	}

	apiKeyCmd.AddCommand(listKeysCmd, createKeyCmd, deleteKeyCmd)

	// Resource commands
	resourceCmd := &cobra.Command{
		Use:   "resource",
		Short: "Manage resources with tenant context",
	}

	// List resources
	listResourcesCmd := &cobra.Command{
		Use:   "list [resource-type]",
		Short: "List resources in tenant context",
		Args:  cobra.ExactArgs(1),
		RunE:  listResources,
	}
	listResourcesCmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID (required)")
	listResourcesCmd.MarkFlagRequired("tenant")

	resourceCmd.AddCommand(listResourcesCmd)

	// Add all commands to root
	rootCmd.AddCommand(tenantCmd, memberCmd, apiKeyCmd, resourceCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func makeRequest(method, endpoint string, body interface{}, headers map[string]string) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, apiURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func printTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	
	// Print headers
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	
	// Print separator
	separators := make([]string, len(headers))
	for i := range separators {
		separators[i] = strings.Repeat("-", len(headers[i]))
	}
	fmt.Fprintln(w, strings.Join(separators, "\t"))
	
	// Print rows
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	
	w.Flush()
}

// Command implementations

func listTenants(cmd *cobra.Command, args []string) error {
	endpoint := "/api/v1/tenants"
	if parent, _ := cmd.Flags().GetString("parent"); parent != "" {
		endpoint += "?parent=" + parent
	}

	data, err := makeRequest("GET", endpoint, nil, nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(data))
		return nil
	}

	var result struct {
		Tenants []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
			Status      string `json:"status"`
		} `json:"tenants"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	headers := []string{"ID", "NAME", "DISPLAY NAME", "TYPE", "STATUS"}
	rows := [][]string{}

	for _, tenant := range result.Tenants {
		rows = append(rows, []string{
			tenant.ID,
			tenant.Name,
			tenant.DisplayName,
			tenant.Type,
			tenant.Status,
		})
	}

	printTable(headers, rows)
	return nil
}

func createTenant(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	displayName, _ := cmd.Flags().GetString("display-name")
	tenantType, _ := cmd.Flags().GetString("type")
	parent, _ := cmd.Flags().GetString("parent")
	maxSwitches, _ := cmd.Flags().GetInt("max-switches")
	maxRouters, _ := cmd.Flags().GetInt("max-routers")

	body := map[string]interface{}{
		"name":         name,
		"display_name": displayName,
		"type":         tenantType,
		"quotas": map[string]int{
			"max_switches": maxSwitches,
			"max_routers":  maxRouters,
		},
	}

	if parent != "" {
		body["parent"] = parent
	}

	data, err := makeRequest("POST", "/api/v1/tenants", body, nil)
	if err != nil {
		return err
	}

	fmt.Println("Tenant created successfully:")
	fmt.Println(string(data))
	return nil
}

func getTenant(cmd *cobra.Command, args []string) error {
	data, err := makeRequest("GET", "/api/v1/tenants/"+args[0], nil, nil)
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func updateTenant(cmd *cobra.Command, args []string) error {
	body := map[string]interface{}{}

	if displayName, _ := cmd.Flags().GetString("display-name"); displayName != "" {
		body["display_name"] = displayName
	}

	quotas := map[string]int{}
	if maxSwitches, _ := cmd.Flags().GetInt("max-switches"); maxSwitches >= 0 {
		quotas["max_switches"] = maxSwitches
	}
	if maxRouters, _ := cmd.Flags().GetInt("max-routers"); maxRouters >= 0 {
		quotas["max_routers"] = maxRouters
	}

	if len(quotas) > 0 {
		body["quotas"] = quotas
	}

	data, err := makeRequest("PUT", "/api/v1/tenants/"+args[0], body, nil)
	if err != nil {
		return err
	}

	fmt.Println("Tenant updated successfully:")
	fmt.Println(string(data))
	return nil
}

func deleteTenant(cmd *cobra.Command, args []string) error {
	_, err := makeRequest("DELETE", "/api/v1/tenants/"+args[0], nil, nil)
	if err != nil {
		return err
	}

	fmt.Println("Tenant marked for deletion")
	return nil
}

func showUsage(cmd *cobra.Command, args []string) error {
	data, err := makeRequest("GET", "/api/v1/tenants/"+args[0]+"/usage", nil, nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(data))
		return nil
	}

	var result struct {
		Usage struct {
			Switches      int `json:"switches"`
			Routers       int `json:"routers"`
			Ports         int `json:"ports"`
			ACLs          int `json:"acls"`
			LoadBalancers int `json:"load_balancers"`
		} `json:"usage"`
		Quotas struct {
			MaxSwitches      int `json:"max_switches"`
			MaxRouters       int `json:"max_routers"`
			MaxPorts         int `json:"max_ports"`
			MaxACLs          int `json:"max_acls"`
			MaxLoadBalancers int `json:"max_load_balancers"`
		} `json:"quotas"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	fmt.Println("Resource Usage:")
	fmt.Printf("  Switches:       %d / %d\n", result.Usage.Switches, result.Quotas.MaxSwitches)
	fmt.Printf("  Routers:        %d / %d\n", result.Usage.Routers, result.Quotas.MaxRouters)
	fmt.Printf("  Ports:          %d / %d\n", result.Usage.Ports, result.Quotas.MaxPorts)
	fmt.Printf("  ACLs:           %d / %d\n", result.Usage.ACLs, result.Quotas.MaxACLs)
	fmt.Printf("  Load Balancers: %d / %d\n", result.Usage.LoadBalancers, result.Quotas.MaxLoadBalancers)
	
	return nil
}

func listMembers(cmd *cobra.Command, args []string) error {
	data, err := makeRequest("GET", "/api/v1/tenants/"+args[0]+"/members", nil, nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(data))
		return nil
	}

	var result struct {
		Members []struct {
			UserID    string `json:"user_id"`
			Role      string `json:"role"`
			CreatedAt string `json:"created_at"`
		} `json:"members"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	headers := []string{"USER ID", "ROLE", "JOINED"}
	rows := [][]string{}

	for _, member := range result.Members {
		rows = append(rows, []string{
			member.UserID,
			member.Role,
			member.CreatedAt,
		})
	}

	printTable(headers, rows)
	return nil
}

func addMember(cmd *cobra.Command, args []string) error {
	role, _ := cmd.Flags().GetString("role")
	
	body := map[string]string{
		"user_id": args[1],
		"role":    role,
	}

	_, err := makeRequest("POST", "/api/v1/tenants/"+args[0]+"/members", body, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Member %s added with role %s\n", args[1], role)
	return nil
}

func removeMember(cmd *cobra.Command, args []string) error {
	_, err := makeRequest("DELETE", "/api/v1/tenants/"+args[0]+"/members/"+args[1], nil, nil)
	if err != nil {
		return err
	}

	fmt.Printf("Member %s removed\n", args[1])
	return nil
}

func listAPIKeys(cmd *cobra.Command, args []string) error {
	data, err := makeRequest("GET", "/api/v1/tenants/"+args[0]+"/api-keys", nil, nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(data))
		return nil
	}

	var result struct {
		Keys []struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Prefix      string   `json:"prefix"`
			Scopes      []string `json:"scopes"`
			LastUsedAt  *string  `json:"last_used_at"`
			ExpiresAt   *string  `json:"expires_at"`
		} `json:"keys"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	headers := []string{"ID", "NAME", "PREFIX", "SCOPES", "LAST USED", "EXPIRES"}
	rows := [][]string{}

	for _, key := range result.Keys {
		lastUsed := "Never"
		if key.LastUsedAt != nil {
			lastUsed = *key.LastUsedAt
		}
		
		expires := "Never"
		if key.ExpiresAt != nil {
			expires = *key.ExpiresAt
		}

		rows = append(rows, []string{
			key.ID,
			key.Name,
			key.Prefix,
			strings.Join(key.Scopes, ","),
			lastUsed,
			expires,
		})
	}

	printTable(headers, rows)
	return nil
}

func createAPIKey(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	scopes, _ := cmd.Flags().GetStringSlice("scopes")
	expiresIn, _ := cmd.Flags().GetInt("expires-in")

	body := map[string]interface{}{
		"name":       name,
		"scopes":     scopes,
		"expires_in": expiresIn,
	}

	data, err := makeRequest("POST", "/api/v1/tenants/"+args[0]+"/api-keys", body, nil)
	if err != nil {
		return err
	}

	var result struct {
		Key     string `json:"key"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	fmt.Println(result.Message)
	fmt.Printf("\nAPI Key: %s\n", result.Key)
	fmt.Println("\nIMPORTANT: Save this key securely. It won't be shown again.")
	
	return nil
}

func deleteAPIKey(cmd *cobra.Command, args []string) error {
	_, err := makeRequest("DELETE", "/api/v1/tenants/"+args[0]+"/api-keys/"+args[1], nil, nil)
	if err != nil {
		return err
	}

	fmt.Println("API key deleted")
	return nil
}

func listResources(cmd *cobra.Command, args []string) error {
	resourceType := args[0]
	endpoint := ""

	switch resourceType {
	case "switches":
		endpoint = "/api/v1/switches"
	case "routers":
		endpoint = "/api/v1/routers"
	case "ports":
		endpoint = "/api/v1/ports"
	case "acls":
		endpoint = "/api/v1/acls"
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}

	headers := map[string]string{
		"X-Tenant-ID": tenantID,
	}

	data, err := makeRequest("GET", endpoint, nil, headers)
	if err != nil {
		return err
	}

	fmt.Printf("Resources in tenant %s:\n", tenantID)
	fmt.Println(string(data))
	return nil
}