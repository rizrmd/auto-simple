package tools

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
)

type WhatsAppInstance struct {
	Client     *whatsmeow.Client
	Downloader *WhatsAppDownloader
	Database   string
	PhoneID    string
	Connected  bool
	mu         sync.RWMutex
}

type WhatsAppManager struct {
	instances map[string]*WhatsAppInstance
	mu        sync.RWMutex
	dbDir     string
}

func NewWhatsAppManager(dbDir string) *WhatsAppManager {
	if dbDir == "" {
		dbDir = "./data"
	}

	// Create database directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Printf("Failed to create database directory: %v", err)
	}

	return &WhatsAppManager{
		instances: make(map[string]*WhatsAppInstance),
		dbDir:     dbDir,
	}
}

func (wm *WhatsAppManager) generateDatabaseName(phoneID string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s/whatsapp_%s_%s.db", wm.dbDir, phoneID, timestamp)
}

func (wm *WhatsAppManager) AddClient(phoneID string) (*WhatsAppInstance, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.instances[phoneID]; exists {
		return nil, fmt.Errorf("client with phoneID %s already exists", phoneID)
	}

	// Generate unique database name
	dbPath := wm.generateDatabaseName(phoneID)

	// Create device store with unique database
	dbLog := waLog.Stdout("DB", "INFO", true)
	deviceStore, err := sqlstore.New(context.Background(), "sqlite3", dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create device store for %s: %w", phoneID, err)
	}

	// Get or create device
	device, err := deviceStore.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device for %s: %w", phoneID, err)
	}

	// Create WhatsApp client
	client := whatsmeow.NewClient(device, waLog.Noop)

	// Create downloader
	downloader := NewWhatsAppDownloader(client)

	instance := &WhatsAppInstance{
		Client:     client,
		Downloader: downloader,
		Database:   dbPath,
		PhoneID:    phoneID,
		Connected:  false,
	}

	wm.instances[phoneID] = instance

	log.Printf("Added WhatsApp client for phoneID: %s with database: %s", phoneID, dbPath)
	return instance, nil
}

func (wm *WhatsAppManager) GetClient(phoneID string) (*WhatsAppInstance, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	instance, exists := wm.instances[phoneID]
	if !exists {
		return nil, fmt.Errorf("client with phoneID %s not found", phoneID)
	}

	return instance, nil
}

func (wm *WhatsAppManager) RemoveClient(phoneID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	instance, exists := wm.instances[phoneID]
	if !exists {
		return fmt.Errorf("client with phoneID %s not found", phoneID)
	}

	// Disconnect if connected
	if instance.Connected {
		instance.Client.Disconnect()
	}

	delete(wm.instances, phoneID)
	log.Printf("Removed WhatsApp client for phoneID: %s", phoneID)
	return nil
}

func (wm *WhatsAppManager) ConnectClient(phoneID string) error {
	instance, err := wm.GetClient(phoneID)
	if err != nil {
		return err
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	if instance.Connected {
		return fmt.Errorf("client %s is already connected", phoneID)
	}

	// Add history sync handlers before connecting
	ctx := context.Background()
	instance.Downloader.AddHistorySyncHandlers(ctx)

	// Add event handlers
	instance.Client.AddEventHandler(func(evt any) {
		switch evt.(type) {
		case *events.Connected:
			instance.mu.Lock()
			instance.Connected = true
			instance.mu.Unlock()
			log.Printf("WhatsApp client %s connected successfully!", phoneID)
		case *events.Disconnected:
			instance.mu.Lock()
			instance.Connected = false
			instance.mu.Unlock()
			log.Printf("WhatsApp client %s disconnected", phoneID)
		case *events.LoggedOut:
			instance.mu.Lock()
			instance.Connected = false
			instance.mu.Unlock()
			log.Printf("WhatsApp client %s was logged out", phoneID)
		}
	})

	// Connect to WhatsApp with QR code handling
	if instance.Client.Store.ID == nil {
		// No ID stored, new login required
		qrChan, _ := instance.Client.GetQRChannel(context.Background())
		err = instance.Client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect client %s for QR login: %w", phoneID, err)
		}

		// Display QR code
		fmt.Printf("\n=== SCAN QR CODE FOR CLIENT: %s ===\n", phoneID)
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan this QR code with WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Printf("Client: %s", phoneID)
				fmt.Println("=====================================")
			}
		}
	} else {
		// Already logged in, just connect
		err = instance.Client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect existing client %s: %w", phoneID, err)
		}
	}

	return nil
}

func (wm *WhatsAppManager) DisconnectClient(phoneID string) error {
	instance, err := wm.GetClient(phoneID)
	if err != nil {
		return err
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	if !instance.Connected {
		return fmt.Errorf("client %s is not connected", phoneID)
	}

	instance.Client.Disconnect()
	instance.Connected = false

	log.Printf("WhatsApp client %s disconnected", phoneID)
	return nil
}

func (wm *WhatsAppManager) ConnectAllClients() error {
	wm.mu.RLock()
	phoneIDs := make([]string, 0, len(wm.instances))
	for phoneID := range wm.instances {
		phoneIDs = append(phoneIDs, phoneID)
	}
	wm.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(phoneIDs))

	for _, phoneID := range phoneIDs {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			if err := wm.ConnectClient(pid); err != nil {
				errChan <- fmt.Errorf("failed to connect client %s: %w", pid, err)
			}
		}(phoneID)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during connection: %v", len(errors), errors)
	}

	return nil
}

func (wm *WhatsAppManager) DisconnectAllClients() error {
	wm.mu.RLock()
	phoneIDs := make([]string, 0, len(wm.instances))
	for phoneID := range wm.instances {
		phoneIDs = append(phoneIDs, phoneID)
	}
	wm.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(phoneIDs))

	for _, phoneID := range phoneIDs {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			if err := wm.DisconnectClient(pid); err != nil {
				errChan <- fmt.Errorf("failed to disconnect client %s: %w", pid, err)
			}
		}(phoneID)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during disconnection: %v", len(errors), errors)
	}

	return nil
}

func (wm *WhatsAppManager) ListClients() []string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	phoneIDs := make([]string, 0, len(wm.instances))
	for phoneID := range wm.instances {
		phoneIDs = append(phoneIDs, phoneID)
	}

	return phoneIDs
}

func (wm *WhatsAppManager) GetClientStatus(phoneID string) (bool, string, error) {
	instance, err := wm.GetClient(phoneID)
	if err != nil {
		return false, "", err
	}

	instance.mu.RLock()
	connected := instance.Connected
	database := instance.Database
	instance.mu.RUnlock()

	return connected, database, nil
}

func (wm *WhatsAppManager) CleanupDatabases() error {
	files, err := filepath.Glob(filepath.Join(wm.dbDir, "whatsapp_*.db"))
	if err != nil {
		return fmt.Errorf("failed to glob database files: %w", err)
	}

	var errors []error
	for _, file := range files {
		if err := os.Remove(file); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove %s: %w", file, err))
		} else {
			log.Printf("Removed database file: %s", file)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during cleanup: %v", len(errors), errors)
	}

	return nil
}
