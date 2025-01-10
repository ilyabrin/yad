package main

// Yad - Yandex.Disk CLI tool

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "v1.0 (alpha)"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: yad [init|version]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("yad version %s\n", version)
	case "init":
		p := tea.NewProgram(initialModel())
		if err := p.Start(); err != nil {
			fmt.Printf("Error running program: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Unknown command. Use 'init' or 'version'")
		os.Exit(1)
	}
}
