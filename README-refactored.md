# AWS Cost Explorer TUI - Refactored

This is a refactored version of the AWS Cost Explorer TUI application with a clean, maintainable architecture that avoids circular imports.

## Project Structure

```
go-cost-explorer/
├── cmd/
│   └── tui/
│       └── main.go              # Entry point - minimal main function
├── internal/
│   ├── app/
│   │   ├── app.go               # Application state and initialization
│   │   └── keybindings.go       # Key handling logic
│   ├── ui/
│   │   ├── components.go        # UI component creation
│   │   ├── layout.go            # Grid setup and layout
│   │   ├── table.go             # Table population and formatting
│   │   └── theme.go             # Theme configuration
│   ├── aws/
│   │   ├── client.go            # AWS client setup
│   │   └── costexplorer.go      # All AWS Cost Explorer API calls
│   ├── cache/
│   │   └── cache.go             # Data caching logic
│   └── types/
│       └── types.go             # Shared types and structs
├── go.mod
└── go.sum
```

## Architecture Benefits

1. **Clear Separation of Concerns**: Each package has a single responsibility
2. **No Circular Imports**: Dependency flow is unidirectional: `cmd` → `app` → `ui`/`aws`/`cache` → `types`
3. **Testability**: Individual components can be easily unit tested
4. **Maintainability**: Easy to modify or extend functionality
5. **Reusability**: AWS functions can be reused in other commands

## Key Design Patterns

- **Dependency Injection**: The cache package accepts a `TablePopulator` interface to avoid importing ui
- **Interface Segregation**: Small, focused interfaces like `TablePopulator`
- **Single Responsibility**: Each package focuses on one aspect of the application
- **Clean Architecture**: Business logic is separated from UI and external dependencies

## Building and Running

```bash
# Build the application
go build ./cmd/tui

# Run the application
./tui
```

## Adding New Features

1. **New AWS API calls**: Add to `internal/aws/costexplorer.go`
2. **New UI components**: Add to `internal/ui/`
3. **New application logic**: Add to `internal/app/`
4. **New data types**: Add to `internal/types/types.go`

This structure makes it easy to extend the application while maintaining clean separation between concerns.
