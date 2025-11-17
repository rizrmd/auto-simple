package main

import (
	"context"
	"fmt"
	"log"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"auto-lmk/pkg/tools"
)

func main() {
	// Initialize WhatsApp client (example - you would normally have proper initialization)
	client := &whatsmeow.Client{} // This would be properly initialized in real usage
	
	// Create downloader with lazy loading
	downloader := tools.NewWhatsAppDownloader(client)
	
	// Example 1: Load existing metadata from file
	ctx := context.Background()
	err := downloader.LoadHistoryMetadata("history_images.json")
	if err != nil {
		log.Printf("No existing metadata found (this is normal for first run): %v", err)
	}
	
	// Example 2: List all available historical images
	images := downloader.ListHistoricalImages()
	fmt.Printf("Found %d historical images available for lazy loading\n", len(images))
	
	for i, img := range images {
		fmt.Printf("%d. %s from %s at %s\n", 
			i+1, img.FileName, img.ChatJID.User, img.Timestamp.Format("2006-01-02 15:04:05"))
	}
	
	// Example 3: Download a specific image when needed
	if len(images) > 0 {
		messageID := images[0].MessageID
		fmt.Printf("\nDownloading image with message ID: %s\n", messageID)
		
		filename, err := downloader.DownloadHistoricalImageByMessageID(ctx, messageID)
		if err != nil {
			log.Printf("Failed to download image: %v", err)
		} else {
			fmt.Printf("Successfully downloaded image to: %s\n", filename)
		}
	}
	
	// Example 4: Save metadata for future use
	err = downloader.SaveHistoryMetadata("history_images.json")
	if err != nil {
		log.Printf("Failed to save metadata: %v", err)
	} else {
		fmt.Println("Saved historical image metadata to history_images.json")
	}
	
	// Example 5: Check if a specific image exists before downloading
	testMessageID := types.MessageID("test_message_id")
	if imageInfo, exists := downloader.GetHistoricalImageInfo(testMessageID); exists {
		fmt.Printf("Image info found for message %s: %s\n", testMessageID, imageInfo.FileName)
		
		// Download it
		filename, err := downloader.DownloadHistoricalImage(ctx, imageInfo)
		if err != nil {
			log.Printf("Failed to download: %v", err)
		} else {
			fmt.Printf("Downloaded: %s\n", filename)
		}
	} else {
		fmt.Printf("No image found for message ID: %s\n", testMessageID)
	}
}