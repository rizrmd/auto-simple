package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
)

// AITools handles AI tool integration for WhatsApp messages
type AITools struct {
	openaiClient openai.Client
	model        string
}

// NewAITools creates a new AI tools handler
func NewAITools(openaiClient openai.Client, model string) *AITools {
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	return &AITools{
		openaiClient: openaiClient,
		model:        model,
	}
}

// validateAndOptimizeImage checks image size and optimizes if necessary
func (at *AITools) validateAndOptimizeImage(imageData []byte, filename string) ([]byte, string, error) {
	// Validate image size
	if err := ValidateImage(imageData); err != nil {
		return nil, "", err
	}

	// Detect image type
	mimeType := DetectImageType(filename, imageData)

	// Resize image for LLM processing (always resize to optimize for LLM)
	resizedData, err := ResizeImageForLLM(imageData, mimeType)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resize image for LLM: %w", err)
	}

	fmt.Printf("Image resized for LLM: %.2fMB -> %.2fMB (%s)\n",
		float64(len(imageData))/1024/1024,
		float64(len(resizedData))/1024/1024,
		mimeType)

	return resizedData, "image/jpeg", nil // Always use JPEG for LLM processing
}

// ProcessImageWithAI handles image processing with multimodal AI
func (at *AITools) ProcessImageWithAI(ctx context.Context, userMessage string, filename string, imageID string, history []openai.ChatCompletionMessageParamUnion, onStatus func(string)) (string, error) {
	fmt.Printf("ProcessImageWithAI: Starting multimodal processing with message: %s, filename: %s, imageID: %s\n", userMessage, filename, imageID)

	// Read image file
	imagePath := fmt.Sprintf("data/%s", filename)
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Validate and potentially optimize image
	optimizedData, mimeType, err := at.validateAndOptimizeImage(imageData, filename)
	if err != nil {
		return "", err
	}

	// Convert image to base64
	base64Image := base64.StdEncoding.EncodeToString(optimizedData)

	// Create enhanced message with image ID reference
	enhancedMessage := userMessage
	if imageID != "" {
		enhancedMessage = fmt.Sprintf("%s\n\n[Image ID: %s]", userMessage, imageID)
	}

	// Add user message with image to history
	updatedHistory := append(history, openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(enhancedMessage),
		openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL:    fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
			Detail: "high",
		}),
	}))

	// Create request with multimodal content using OpenAI Go library
	req := openai.ChatCompletionNewParams{
		Model:       at.model,
		Messages:    updatedHistory,
		MaxTokens:   openai.Int(500),
		Temperature: openai.Float(0.7),
	}

	fmt.Printf("ProcessImageWithAI: Sending multimodal request to AI model: %s\n", at.model)
	resp, err := at.openaiClient.Chat.Completions.New(ctx, req)
	if err != nil {
		return "", fmt.Errorf("multimodal AI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "Maaf, saya tidak dapat merespons gambar tersebut saat ini.", nil
	}

	response := strings.TrimSpace(resp.Choices[0].Message.Content)

	if onStatus != nil {
		onStatus("⚡ Menyiapkan respons...")
	}

	return response, nil
}

// ProcessTextWithAI handles text processing with optional referenced images
func (at *AITools) ProcessTextWithAI(ctx context.Context, userMessage string, referencedImages []map[string]string, history []openai.ChatCompletionMessageParamUnion, onStatus func(string)) (string, error) {
	fmt.Printf("ProcessTextWithAI: Starting processing with message: %s, referenced images: %d\n", userMessage, len(referencedImages))

	// Create enhanced message with image references
	enhancedMessage := userMessage
	if len(referencedImages) > 0 {
		enhancedMessage += "\n\nGambar yang dirujuk:"
		for _, img := range referencedImages {
			enhancedMessage += fmt.Sprintf("\n[Image ID: %s]", img["id"])
		}
	}

	// Build message content parts
	var contentParts []openai.ChatCompletionContentPartUnionParam
	contentParts = append(contentParts, openai.TextContentPart(enhancedMessage))

	// Add referenced images
	for _, img := range referencedImages {
		imagePath := fmt.Sprintf("data/%s", img["filename"])
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			fmt.Printf("Failed to read referenced image %s: %v\n", img["id"], err)
			continue
		}

		// Validate and optimize image
		optimizedData, mimeType, err := at.validateAndOptimizeImage(imageData, img["filename"])
		if err != nil {
			fmt.Printf("Failed to optimize referenced image %s: %v\n", img["id"], err)
			continue
		}

		// Convert to base64
		base64Image := base64.StdEncoding.EncodeToString(optimizedData)

		contentParts = append(contentParts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
			URL:    fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image),
			Detail: "high",
		}))
	}

	// Add user message with content to history
	updatedHistory := append(history, openai.UserMessage(contentParts))

	// Create request with multimodal content
	req := openai.ChatCompletionNewParams{
		Model:       at.model,
		Messages:    updatedHistory,
		MaxTokens:   openai.Int(500),
		Temperature: openai.Float(0.7),
	}

	resp, err := at.openaiClient.Chat.Completions.New(ctx, req)
	if err != nil {
		return "", fmt.Errorf("text AI API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "Maaf, saya tidak dapat merespons pesan tersebut saat ini.", nil
	}

	response := strings.TrimSpace(resp.Choices[0].Message.Content)
	return response, nil
}
