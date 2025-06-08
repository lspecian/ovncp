// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// This is a helper to download OVN schema if needed
	schemaPath := "../schema/ovn-nb.ovsschema"
	
	// Check if schema exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		fmt.Println("OVN schema not found, downloading...")
		
		// Create schema directory
		schemaDir := filepath.Dir(schemaPath)
		if err := os.MkdirAll(schemaDir, 0755); err != nil {
			log.Fatal("Failed to create schema directory:", err)
		}
		
		// Download schema from OVN repository
		cmd := exec.Command("curl", "-L", "-o", schemaPath,
			"https://raw.githubusercontent.com/ovn-org/ovn/main/ovn-nb.ovsschema")
		
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Fatalf("Failed to download schema: %v\nOutput: %s", err, output)
		}
		
		fmt.Println("Schema downloaded successfully")
	}
	
	// Verify schema is valid JSON
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatal("Failed to read schema:", err)
	}
	
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		log.Fatal("Invalid schema JSON:", err)
	}
	
	fmt.Printf("Schema loaded successfully. Version: %v\n", schema["version"])
	fmt.Println("\nTo generate models, run:")
	fmt.Println("go run github.com/ovn-org/libovsdb/cmd/modelgen \\")
	fmt.Println("  -p nbdb \\")
	fmt.Println("  -o ../nbdb \\")
	fmt.Println("  ../schema/ovn-nb.ovsschema")
}