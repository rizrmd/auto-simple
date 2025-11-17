# WhatsApp Historical Image Lazy Loading

This implementation changes the behavior from eagerly downloading all historical images during history sync to lazily downloading them only when needed.

## Changes Made

### Before (Eager Loading)
- All historical images were downloaded immediately during history sync processing
- Large memory usage and storage consumption at startup
- No way to selectively download specific images

### After (Lazy Loading)
- Only metadata is stored during history sync processing
- Images are downloaded only when explicitly requested
- Efficient memory usage and storage consumption
- Support for metadata persistence (save/load to/from JSON)

## New API Methods

### Core Lazy Loading Methods
```go
// Download a specific historical image by message ID
filename, err := downloader.DownloadHistoricalImageByMessageID(ctx, messageID)

// Get metadata for a historical image without downloading it
imageInfo, exists := downloader.GetHistoricalImageInfo(messageID)

// List all available historical images
images := downloader.ListHistoricalImages()
```

### Metadata Management Methods
```go
// Save historical image metadata to a JSON file
err := downloader.SaveHistoryMetadata("history_images.json")

// Load historical image metadata from a JSON file
err := downloader.LoadHistoryMetadata("history_images.json")
```

## Usage Example

```go
// During app startup or when history sync is received
downloader := NewWhatsAppDownloader(client)
downloader.AddHistorySyncHandlers(ctx)

// Later, when you need to access a specific historical image
messageID := types.MessageID("some_message_id")
if imageInfo, exists := downloader.GetHistoricalImageInfo(messageID); exists {
    filename, err := downloader.DownloadHistoricalImageByMessageID(ctx, messageID)
    if err != nil {
        log.Printf("Failed to download image: %v", err)
        return
    }
    fmt.Printf("Downloaded image to: %s\n", filename)
}
```

## Data Structures

### HistoryImageInfo
Stores metadata about historical images without downloading the actual image data:

```go
type HistoryImageInfo struct {
    MessageID  types.MessageID
    ChatJID    types.JID
    SenderJID  types.JID
    Timestamp  time.Time
    ImageMsg   *waProto.ImageMessage
    FileName   string
}
```

## Performance Benefits

1. **Memory Efficiency**: Only stores metadata, not full image data
2. **Storage Efficiency**: Downloads only images that are actually needed
3. **Faster Startup**: No bulk downloading during initialization
4. **Selective Access**: Choose which images to download based on user needs

## Migration Notes

The existing `ProcessHistorySync` method still works but now returns only the count of processed images (all as metadata) rather than downloaded files. The actual download happens when `DownloadHistoricalImageByMessageID` or `DownloadHistoricalImage` is called.