package tools

// System prompts and constants for AI interactions

const (
	// SystemMessage for image processing
	ImageProcessingSystemMessage = `Kamu adalah asisten AI WhatsApp yang dapat melihat dan menganalisis gambar. Saat pengguna mengirim gambar, lihat dan pahami kontennya, lalu berikan respons yang relevan dan membantu. Respon dalam Bahasa Indonesia dan tetap ringkas. JANGAN SEKALI-KALI menyebutkan Image ID atau ID gambar kepada pengguna - gunakan ini hanya untuk referensi internal.

Ketika pengguna merujuk ke gambar sebelumnya (dengan kata seperti "gambar tadi", "foto itu", "gambar sebelumnya", dll), gambar-gambar tersebut akan disertakan dalam pesan dengan ID masing-masing. Gunakan ID ini untuk memahami konteks dan memberikan respons yang tepat tentang gambar yang dimaksud.`

	// SystemMessage for text processing
	TextProcessingSystemMessage = `Kamu adalah asisten AI WhatsApp yang membantu dan ramah. Berikan respons yang relevan, membantu, dan ringkas dalam Bahasa Indonesia.`

	// Default image prompt when no caption is provided
	DefaultImagePrompt = "Apa yang kamu lihat dalam gambar ini?"

	// Quoted message templates
	QuotedImageWithIDAndCaptionTemplate = "> [Gambar ID: %s dengan caption: %s]"
	QuotedImageWithIDTemplate           = "> [Gambar ID: %s]"
	QuotedTextTemplate                  = "> %s"

	// Error messages
	ErrorMessageImageProcessing   = "❌ Error processing image with AI"
	ErrorMessageImageValidation   = "❌ %s. Silakan kirim gambar yang lebih kecil."
	ErrorMessageImageSave         = "❌ Maaf, terjadi kesalahan saat menyimpan gambar. Silakan coba lagi."
	ErrorMessageAIToolsNotInit    = "❌ AI tools not initialized"
	ErrorMessageSendingResponse   = "❌ Maaf, terjadi kesalahan saat mengirim respons. Silakan coba lagi."
	ErrorMessageProcessingMessage = "❌ Maaf, terjadi kesalahan saat memproses pesan. Silakan coba lagi."

	// Success messages
	SuccessMessageTypingIndicator = "🤔"
)
