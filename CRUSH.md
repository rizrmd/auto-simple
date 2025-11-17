# CRUSH.md - WhatsApp Multi-Client Manager

Essential information for agents working with this Go-based WhatsApp management application.

## Project Overview

A Go CLI application for managing multiple WhatsApp client instances simultaneously. Uses whatsmeow library for WhatsApp Web functionality and provides an interactive menu-driven interface for client management.

## Build & Run Commands

```bash
# Build the application
go build -o whatsapp-manager main.go

# Run directly from source
go run main.go

# Alternative entry point (cmd directory)
go run cmd/whatsapp-manager/main.go

# Test (currently no test files exist)
go test ./...

# Clean build artifacts
go clean
```

## Code Organization

```
├── main.go                    # Primary entry point
├── cmd/whatsapp-manager/      # Alternative entry point
├── pkg/
│   ├── cli/                   # CLI menu interface
│   │   └── menu.go           # Interactive menu logic
│   ├── tools/                 # Core business logic
│   │   ├── ai_tools.go       # AI integration (OpenAI)
│   │   ├── image_utils.go    # Image processing utilities
│   │   ├── prompts.go        # AI prompt templates
│   │   ├── whatsapp.go       # WhatsApp utilities
│   │   └── whatsapp_manager.go # Multi-client management
│   └── whatsapp/             # WhatsApp service layer (empty)
├── data/                      # Database storage directory (gitignored)
└── databases/                 # Alternative storage directory
```

## Dependencies & Key Libraries

- **go.mau.fi/whatsmeow**: WhatsApp Web client library
- **github.com/openai/openai-go**: OpenAI API integration
- **github.com/mattn/go-sqlite3**: SQLite database driver
- **github.com/mdp/qrterminal**: QR code terminal display
- **github.com/joho/godotenv**: Environment variable management

## Core Concepts

### Multi-Client Architecture
- Each WhatsApp client runs in its own instance with separate SQLite database
- Databases are timestamped: `whatsapp_{phoneID}_{timestamp}.db`
- Thread-safe operations using sync.RWMutex for concurrent access

### Client Management Workflow
1. **Add Client**: Creates new instance with unique database
2. **Connect**: QR code scanning for first-time authentication
3. **Manage**: Bulk operations (connect/disconnect all clients)
4. **Monitor**: Real-time status tracking with event handlers

### Event Handling
- Connected/Disconnected/LoggedOut events are captured
- Status updates maintain thread-safe connection state
- History sync handlers manage message persistence

## Database Schema

Each client uses a separate SQLite database containing:
- Device authentication data
- Message history
- Contact information
- Session state

## Configuration

- Environment variables loaded from `.env` file
- Default database directory: `./data`
- OpenAI model defaults to `gpt-3.5-turbo`
- Database path: `file:{path}?_foreign_keys=on`

## UI/CLI Patterns

### Menu System
- Clear screen between operations (`\033[H\033[2J`)
- Numbered options (1-9) with emoji indicators
- Indonesian language interface
- Pause before returning to main menu
- Input validation with user-friendly error messages

### Status Indicators
- 🟢 Connected
- 🔴 Disconnected
- ❌ Error state

## Code Style Conventions

- Go standard formatting (`go fmt`)
- Mixed Indonesian/English for user interface
- Struct methods with receiver names (`m *Menu`, `wm *WhatsAppManager`)
- Error wrapping with context: `fmt.Errorf("failed to %s: %w", operation, err)`

## Known Issues & Gotchas

1. **CLI Infinite Loop**: Menu runs indefinitely until explicit exit (option 0)
   - No timeout mechanism - can run forever
   - Invalid inputs loop back to main menu with error message
   - Testing with `echo` and `timeout` recommended to avoid infinite loops

2. **Database Isolation**: Each client uses separate database file
   - No cross-client data sharing
   - Timestamped naming prevents conflicts
   - Default database directory: `./data` (created automatically)

3. **SQLite Driver Issues**: Fixed database connection issue
   - Requires proper sqlstore.New parameters: `sqlstore.New(ctx, "sqlite3", dbPath+"?_foreign_keys=on", logger)`
   - The "file:" prefix is automatically added by the library

## Testing & Development Notes

```bash
# Test CLI safely (with timeout to avoid infinite loops)
echo "0" | timeout 2s go run main.go  # Immediately exit
echo "1" | timeout 2s go run main.go  # List clients then timeout
echo "2" | timeout 5s go run main.go  # Test add client (empty input)

# Build and run binary
go build -o whatsapp-manager main.go
./whatsapp-manager
```

## Testing

- No test files currently exist in the codebase
- Integration testing would require:
  - Mock WhatsApp connections
  - Database test fixtures
  - CLI input simulation

## Deployment Notes

- Binary name: `whatsapp-manager`
- Requires data directory permissions
- SQLite databases created automatically
- QR codes display in terminal (requires compatible terminal)

## Environment Setup

```bash
# Required Go version: 1.25.3+
go version

# Dependencies are managed automatically
go mod tidy

# Create .env file if needed (for OpenAI API)
echo "OPENAI_API_KEY=your_key_here" > .env
```

## Memory File Instructions

- This project has a `.specify/` directory with development templates
- Constitution exists but is template-only
- No active development workflow rules enforced