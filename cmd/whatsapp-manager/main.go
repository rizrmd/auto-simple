package main

import (
	"log"

	"auto-lmk/pkg/cli"
	"auto-lmk/pkg/tools"
)

func main() {
	// Create WhatsApp manager with custom database directory
	manager := tools.NewWhatsAppManager("./data")

	// Create and run CLI menu
	menu := cli.NewMenu(manager)

	log.Println("📱 WhatsApp Multi-Client Manager")
	log.Println("================================")
	log.Println("🔧 Manage multiple WhatsApp numbers with ease")
	log.Println("")

	menu.ShowMainMenu()
}
