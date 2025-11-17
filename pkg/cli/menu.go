package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"auto-lmk/pkg/tools"
)

type Menu struct {
	manager *tools.WhatsAppManager
	reader  *bufio.Reader
}

func NewMenu(manager *tools.WhatsAppManager) *Menu {
	return &Menu{
		manager: manager,
		reader:  bufio.NewReader(os.Stdin),
	}
}

func (m *Menu) ShowMainMenu() {
	for {
		m.clearScreen()
		m.printHeader()
		m.printOptions()

		choice := m.getInput("Pilih menu (1-9): ")

		switch choice {
		case "1":
			m.listClients()
		case "2":
			m.addClient()
		case "3":
			m.removeClient()
		case "4":
			m.connectClient()
		case "5":
			m.disconnectClient()
		case "6":
			m.connectAllClients()
		case "7":
			m.disconnectAllClients()
		case "8":
			m.showClientStatus()
		case "9":
			m.cleanupDatabases()
		case "0":
			fmt.Println("Keluar dari program...")
			return
		default:
			fmt.Println("Pilihan tidak valid. Silakan coba lagi.")
			m.pause()
		}
	}
}

func (m *Menu) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (m *Menu) printHeader() {
	fmt.Println("========================================")
	fmt.Println("   WHATSAPP MULTI-CLIENT MANAGER")
	fmt.Println("========================================")
	fmt.Println()
}

func (m *Menu) printOptions() {
	fmt.Println("Menu:")
	fmt.Println("1. 📋 List Semua Client")
	fmt.Println("2. ➕ Tambah Client Baru")
	fmt.Println("3. 🗑️  Hapus Client")
	fmt.Println("4. 🔗 Connect Client (Scan QR)")
	fmt.Println("5. 🔌 Disconnect Client")
	fmt.Println("6. 🔗 Connect Semua Client")
	fmt.Println("7. 🔌 Disconnect Semua Client")
	fmt.Println("8. 📊 Lihat Status Client")
	fmt.Println("9. 🧹 Cleanup Database")
	fmt.Println("0. 🚪 Keluar")
	fmt.Println()
}

func (m *Menu) getInput(prompt string) string {
	fmt.Print(prompt)
	input, _ := m.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (m *Menu) pause() {
	fmt.Println("\nTekan Enter untuk melanjutkan...")
	m.reader.ReadString('\n')
}

func (m *Menu) listClients() {
	m.clearScreen()
	fmt.Println("=== DAFTAR CLIENT ===")

	clients := m.manager.ListClients()
	if len(clients) == 0 {
		fmt.Println("📭 Belum ada client yang terdaftar.")
		fmt.Println("💡 Gunakan menu 'Tambah Client Baru' untuk menambah client")
	} else {
		fmt.Printf("📱 Total Client: %d\n\n", len(clients))
		for i, clientName := range clients {
			connected, dbPath, err := m.manager.GetClientStatus(clientName)
			if err != nil {
				fmt.Printf("%d. 📱 %s - ❌ Error: %v\n", i+1, clientName, err)
				continue
			}

			status := "🔴 Disconnected"
			if connected {
				status = "🟢 Connected"
			}

			fmt.Printf("%d. 📱 %s\n", i+1, clientName)
			fmt.Printf("   Status: %s\n", status)
			fmt.Printf("   Database: %s\n", dbPath)
			fmt.Println()
		}
	}

	m.pause()
}

func (m *Menu) addClient() {
	m.clearScreen()
	fmt.Println("=== TAMBAH CLIENT BARU ===")

	clientName := m.getInput("Masukkan Nama Client (contoh: Business1, Personal, etc): ")
	if clientName == "" {
		fmt.Println("Nama client tidak boleh kosong!")
		m.pause()
		return
	}

	instance, err := m.manager.AddClient(clientName)
	if err != nil {
		fmt.Printf("Gagal menambah client: %v\n", err)
	} else {
		fmt.Printf("✅ Client '%s' berhasil ditambahkan!\n", clientName)
		fmt.Printf("📁 Database: %s\n", instance.Database)
		fmt.Println("\n💡 Tips: Gunakan menu 'Connect Client' untuk scan QR code")
	}

	m.pause()
}

func (m *Menu) removeClient() {
	m.clearScreen()
	fmt.Println("=== HAPUS CLIENT ===")

	clients := m.manager.ListClients()
	if len(clients) == 0 {
		fmt.Println("Belum ada client yang terdaftar.")
		m.pause()
		return
	}

	fmt.Println("Pilih client yang akan dihapus:")
	for i, phoneID := range clients {
		fmt.Printf("%d. %s\n", i+1, phoneID)
	}

	choice := m.getInput("Pilih nomor (0 untuk batal): ")

	if choice == "0" {
		return
	}

	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(clients) {
		fmt.Println("Pilihan tidak valid!")
		m.pause()
		return
	}

	phoneID := clients[index-1]

	confirm := m.getInput(fmt.Sprintf("Yakin ingin menghapus client %s? (y/N): ", phoneID))
	if strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes" {
		err := m.manager.RemoveClient(phoneID)
		if err != nil {
			fmt.Printf("Gagal menghapus client: %v\n", err)
		} else {
			fmt.Printf("Client %s berhasil dihapus!\n", phoneID)
		}
	} else {
		fmt.Println("Penghapusan dibatalkan.")
	}

	m.pause()
}

func (m *Menu) connectClient() {
	m.clearScreen()
	fmt.Println("=== CONNECT CLIENT ===")

	clients := m.manager.ListClients()
	if len(clients) == 0 {
		fmt.Println("Belum ada client yang terdaftar.")
		fmt.Println("💡 Tips: Gunakan menu 'Tambah Client Baru' terlebih dahulu")
		m.pause()
		return
	}

	fmt.Println("Pilih client yang akan di-connect:")
	for i, clientName := range clients {
		connected, _, _ := m.manager.GetClientStatus(clientName)
		status := "🔴 Disconnected"
		if connected {
			status = "🟢 Connected"
		}
		fmt.Printf("%d. %s (%s)\n", i+1, clientName, status)
	}

	choice := m.getInput("Pilih nomor (0 untuk batal): ")

	if choice == "0" {
		return
	}

	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(clients) {
		fmt.Println("❌ Pilihan tidak valid!")
		m.pause()
		return
	}

	clientName := clients[index-1]

	fmt.Printf("\n🔄 Menghubungkan client '%s'...\n", clientName)
	fmt.Println("📱 Siapkan WhatsApp di HP untuk scan QR code")
	fmt.Println()

	err = m.manager.ConnectClient(clientName)
	if err != nil {
		fmt.Printf("❌ Gagal connect client: %v\n", err)
	} else {
		fmt.Printf("✅ Client '%s' berhasil di-connect!\n", clientName)
	}

	m.pause()
}

func (m *Menu) disconnectClient() {
	m.clearScreen()
	fmt.Println("=== DISCONNECT CLIENT ===")

	clients := m.manager.ListClients()
	if len(clients) == 0 {
		fmt.Println("Belum ada client yang terdaftar.")
		m.pause()
		return
	}

	fmt.Println("Pilih client yang akan di-disconnect:")
	for i, phoneID := range clients {
		connected, _, _ := m.manager.GetClientStatus(phoneID)
		status := "Disconnected"
		if connected {
			status = "Connected"
		}
		fmt.Printf("%d. %s (%s)\n", i+1, phoneID, status)
	}

	choice := m.getInput("Pilih nomor (0 untuk batal): ")

	if choice == "0" {
		return
	}

	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(clients) {
		fmt.Println("Pilihan tidak valid!")
		m.pause()
		return
	}

	phoneID := clients[index-1]

	fmt.Printf("Memutuskan koneksi client %s...\n", phoneID)
	err = m.manager.DisconnectClient(phoneID)
	if err != nil {
		fmt.Printf("Gagal disconnect client: %v\n", err)
	} else {
		fmt.Printf("Client %s berhasil di-disconnect!\n", phoneID)
	}

	m.pause()
}

func (m *Menu) connectAllClients() {
	m.clearScreen()
	fmt.Println("=== CONNECT SEMUA CLIENT ===")

	fmt.Println("Menghubungkan semua client...")
	err := m.manager.ConnectAllClients()
	if err != nil {
		fmt.Printf("Terjadi error saat connect: %v\n", err)
	} else {
		fmt.Println("Semua client berhasil di-connect!")
	}

	m.pause()
}

func (m *Menu) disconnectAllClients() {
	m.clearScreen()
	fmt.Println("=== DISCONNECT SEMUA CLIENT ===")

	fmt.Println("Memutuskan koneksi semua client...")
	err := m.manager.DisconnectAllClients()
	if err != nil {
		fmt.Printf("Terjadi error saat disconnect: %v\n", err)
	} else {
		fmt.Println("Semua client berhasil di-disconnect!")
	}

	m.pause()
}

func (m *Menu) showClientStatus() {
	m.clearScreen()
	fmt.Println("=== STATUS CLIENT ===")

	clients := m.manager.ListClients()
	if len(clients) == 0 {
		fmt.Println("Belum ada client yang terdaftar.")
	} else {
		for _, phoneID := range clients {
			connected, dbPath, err := m.manager.GetClientStatus(phoneID)
			if err != nil {
				fmt.Printf("❌ %s - Error: %v\n", phoneID, err)
				continue
			}

			status := "🔴 Disconnected"
			if connected {
				status = "🟢 Connected"
			}

			fmt.Printf("📱 %s\n", phoneID)
			fmt.Printf("   Status: %s\n", status)
			fmt.Printf("   Database: %s\n", dbPath)
			fmt.Println()
		}
	}

	m.pause()
}

func (m *Menu) cleanupDatabases() {
	m.clearScreen()
	fmt.Println("=== CLEANUP DATABASE ===")

	confirm := m.getInput("Yakin ingin menghapus semua database? (y/N): ")
	if strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes" {
		fmt.Println("Menghapus database...")
		err := m.manager.CleanupDatabases()
		if err != nil {
			fmt.Printf("Gagal cleanup database: %v\n", err)
		} else {
			fmt.Println("Database berhasil dibersihkan!")
		}
	} else {
		fmt.Println("Cleanup dibatalkan.")
	}

	m.pause()
}
