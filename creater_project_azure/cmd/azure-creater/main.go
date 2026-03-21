package main

import (
	"fmt"
	"os"

	generator "github.com/Payel-git-ol/azure/creater_project_azure"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: azure-creater <config.yaml>")
		fmt.Println("Example: azure-creater create-api-azure.yaml")
		os.Exit(1)
	}

	configPath := os.Args[1]

	fmt.Println("🔧 Azure Project Creator")
	fmt.Println("========================")
	fmt.Printf("📄 Config: %s\n\n", configPath)

	// Создаём генератор
	gen, err := generator.NewGenerator(configPath)
	if err != nil {
		fmt.Printf("❌ Error creating generator: %v\n", err)
		os.Exit(1)
	}

	// Генерируем проект
	if err := gen.Generate(); err != nil {
		fmt.Printf("❌ Error generating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✨ Next steps:")
	fmt.Println("   1. Review generated files")
	fmt.Println("   2. Implement your logic in // TODO sections")
	fmt.Println("   3. Run: go mod tidy")
	fmt.Println("   4. Run: go run cmd/app/main.go")
}
