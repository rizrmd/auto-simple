package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"log"
	"os"
	"sync"
	"time"
)

type WhatsAppDownloader struct {
	client            *whatsmeow.Client
	historyImages     map[string]HistoryImageInfo
	historyImagesMutex sync.RWMutex
}

func NewWhatsAppDownloader(client *whatsmeow.Client) *WhatsAppDownloader {
	return &WhatsAppDownloader{
		client:        client,
		historyImages: make(map[string]HistoryImageInfo),
	}
}

func (wd *WhatsAppDownloader) DownloadImage(ctx context.Context, msgInfo types.MessageInfo, imgMsg *waProto.ImageMessage) ([]byte, error) {
	if wd.client == nil {
		return nil, fmt.Errorf("WhatsApp client not initialized")
	}

	// Download the image data
	data, err := wd.client.Download(ctx, imgMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// data is already []byte, return directly
	return data, nil
}

func (wd *WhatsAppDownloader) GetImageCaption(imgMsg *waProto.ImageMessage) string {
	if imgMsg.Caption != nil {
		return *imgMsg.Caption
	}
	return ""
}

func (wd *WhatsAppDownloader) GetImageType(imgMsg *waProto.ImageMessage) string {
	if imgMsg.Mimetype != nil {
		return *imgMsg.Mimetype
	}
	return "image/jpeg" // Default fallback
}

// RequestHistorySync requests additional history from the user's primary device
func (wd *WhatsAppDownloader) RequestHistorySync(ctx context.Context, lastKnownMessageInfo *types.MessageInfo, count int) error {
	if wd.client == nil {
		return fmt.Errorf("WhatsApp client not initialized")
	}

	// Build history sync request message
	historySyncMsg := wd.client.BuildHistorySyncRequest(lastKnownMessageInfo, count)
	if historySyncMsg == nil {
		return fmt.Errorf("failed to build history sync request")
	}

	// Send the history sync request with peer flag
	_, err := wd.client.SendMessage(ctx, types.EmptyJID, historySyncMsg, whatsmeow.SendRequestExtra{Peer: true})
	if err != nil {
		return fmt.Errorf("failed to send history sync request: %w", err)
	}

	return nil
}

// AddHistorySyncHandlers adds event handlers for history sync notifications.
// This now processes history sync data lazily - it only stores metadata about historical images
// without downloading them. Images are downloaded on-demand using DownloadHistoricalImageByMessageID().
func (wd *WhatsAppDownloader) AddHistorySyncHandlers(ctx context.Context) {
	if wd.client == nil {
		log.Printf("WhatsApp client not initialized, cannot add history sync handlers")
		return
	}

		wd.client.AddEventHandler(func(evt any) {
		if v, ok := evt.(*events.HistorySync); ok {
			// The event fires after the history sync blob has been downloaded and decrypted.
			fmt.Printf("History sync event received. Processing %d conversations for image metadata...\n", len(v.Data.Conversations))
			_, err := wd.processHistorySyncData(ctx, v.Data)
			if err != nil {
				log.Printf("Failed to process history sync data: %v", err)
				return
			}
			fmt.Printf("Successfully processed history sync. Images will be downloaded on-demand.\n")
		}
	})
}

// HistoryImageInfo stores metadata about historical images without downloading them
type HistoryImageInfo struct {
	MessageID  types.MessageID
	ChatJID    types.JID
	SenderJID  types.JID
	Timestamp  time.Time
	ImageMsg   *waProto.ImageMessage
	FileName   string
}

// processHistorySyncData processes the parsed history sync data and stores image metadata for lazy loading
func (wd *WhatsAppDownloader) processHistorySyncData(ctx context.Context, historySync *waHistorySync.HistorySync) ([]string, error) {
	if wd.client == nil {
		return nil, fmt.Errorf("WhatsApp client not initialized")
	}

	var downloadedFiles []string

	// Process conversations in the history sync
	for _, conversation := range historySync.Conversations {
		if conversation.Messages == nil {
			continue
		}

		conversationID := conversation.GetID()
		jid, err := types.ParseJID(conversationID)
		if err != nil {
			fmt.Printf("Warning: failed to parse conversation ID %s: %v\n", conversationID, err)
			continue
		}

		// Process each message in the conversation
		for _, historyMsg := range conversation.Messages {
			webMsg := historyMsg.GetMessage()
			if webMsg == nil || webMsg.Message == nil {
				continue
			}

			// Check if the message contains an image
			if webMsg.Message.GetImageMessage() != nil {
				imgMsg := webMsg.Message.GetImageMessage()

				// Create a MessageInfo for the historical message
				msgInfo := types.MessageInfo{
					ID:        types.MessageID(webMsg.GetKey().GetID()),
					Timestamp: time.Unix(int64(webMsg.GetMessageTimestamp()), 0),
				}
				msgInfo.Chat = jid
				msgInfo.Sender = jid

				// Store image metadata for lazy loading instead of downloading immediately
				timestamp := time.Unix(int64(webMsg.GetMessageTimestamp()), 0)
				filename := fmt.Sprintf("historical_%s_%s.jpg",
					timestamp.Format("20060102_150405"),
					webMsg.GetKey().GetID())

				imageInfo := HistoryImageInfo{
					MessageID: msgInfo.ID,
					ChatJID:   jid,
					SenderJID: jid,
					Timestamp: timestamp,
					ImageMsg:  imgMsg,
					FileName:  filename,
				}

				// Store the image metadata for later lazy loading
				wd.historyImagesMutex.Lock()
				wd.historyImages[string(msgInfo.ID)] = imageInfo
				wd.historyImagesMutex.Unlock()

				fmt.Printf("Found historical image metadata: %s (not downloaded yet)\n", imageInfo.FileName)
			}
		}
	}

	return downloadedFiles, nil
}

// GetHistoricalImageInfo retrieves metadata for a historical image by message ID
func (wd *WhatsAppDownloader) GetHistoricalImageInfo(messageID types.MessageID) (HistoryImageInfo, bool) {
	wd.historyImagesMutex.RLock()
	defer wd.historyImagesMutex.RUnlock()
	
	imageInfo, exists := wd.historyImages[string(messageID)]
	return imageInfo, exists
}

// ListHistoricalImages returns a list of all historical image metadata
func (wd *WhatsAppDownloader) ListHistoricalImages() []HistoryImageInfo {
	wd.historyImagesMutex.RLock()
	defer wd.historyImagesMutex.RUnlock()
	
	images := make([]HistoryImageInfo, 0, len(wd.historyImages))
	for _, imageInfo := range wd.historyImages {
		images = append(images, imageInfo)
	}
	return images
}

// SaveHistoryMetadata saves the historical image metadata to a JSON file
func (wd *WhatsAppDownloader) SaveHistoryMetadata(filename string) error {
	wd.historyImagesMutex.RLock()
	defer wd.historyImagesMutex.RUnlock()
	
	data, err := json.MarshalIndent(wd.historyImages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history metadata: %w", err)
	}
	
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save history metadata to %s: %w", filename, err)
	}
	
	return nil
}

// LoadHistoryMetadata loads historical image metadata from a JSON file
func (wd *WhatsAppDownloader) LoadHistoryMetadata(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read history metadata from %s: %w", filename, err)
	}
	
	var loadedImages map[string]HistoryImageInfo
	err = json.Unmarshal(data, &loadedImages)
	if err != nil {
		return fmt.Errorf("failed to unmarshal history metadata: %w", err)
	}
	
	wd.historyImagesMutex.Lock()
	wd.historyImages = loadedImages
	wd.historyImagesMutex.Unlock()
	
	return nil
}

// DownloadHistoricalImageByMessageID downloads a historical image by its message ID
func (wd *WhatsAppDownloader) DownloadHistoricalImageByMessageID(ctx context.Context, messageID types.MessageID) (string, error) {
	imageInfo, exists := wd.GetHistoricalImageInfo(messageID)
	if !exists {
		return "", fmt.Errorf("historical image with message ID %s not found", messageID)
	}
	
	return wd.DownloadHistoricalImage(ctx, imageInfo)
}

// DownloadHistoricalImage downloads a specific historical image using its metadata
func (wd *WhatsAppDownloader) DownloadHistoricalImage(ctx context.Context, imageInfo HistoryImageInfo) (string, error) {
	if wd.client == nil {
		return "", fmt.Errorf("WhatsApp client not initialized")
	}

	// Check if file already exists
	if _, err := os.Stat(imageInfo.FileName); err == nil {
		fmt.Printf("Historical image already exists: %s\n", imageInfo.FileName)
		return imageInfo.FileName, nil
	}

	// Create MessageInfo for downloading
	msgInfo := types.MessageInfo{
		ID:        imageInfo.MessageID,
		Timestamp: imageInfo.Timestamp,
	}
	msgInfo.Chat = imageInfo.ChatJID
	msgInfo.Sender = imageInfo.SenderJID

	// Download the image
	imageData, err := wd.DownloadImage(ctx, msgInfo, imageInfo.ImageMsg)
	if err != nil {
		return "", fmt.Errorf("failed to download historical image %s: %w", imageInfo.MessageID, err)
	}

	// Save the image to a file
	err = os.WriteFile(imageInfo.FileName, imageData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save historical image %s: %w", imageInfo.FileName, err)
	}

	fmt.Printf("Downloaded historical image on demand: %s\n", imageInfo.FileName)
	return imageInfo.FileName, nil
}

// ProcessHistorySync processes a history sync notification and stores historical image metadata
func (wd *WhatsAppDownloader) ProcessHistorySync(ctx context.Context, notif *waProto.HistorySyncNotification) ([]string, error) {
	if wd.client == nil {
		return nil, fmt.Errorf("WhatsApp client not initialized")
	}

	// Download and parse the history sync blob
	historySync, err := wd.client.DownloadHistorySync(ctx, notif, false)
	if err != nil {
		return nil, fmt.Errorf("failed to download history sync: %w", err)
	}

	return wd.processHistorySyncData(ctx, historySync)
}
