# Quay

Quay is a powerful Docker Compose wrapper that lets you selectively run services and override port mappings without modifying your compose files. It simplifies working with complex multi-service applications by allowing you to focus only on the parts you need while maintaining full compatibility with all Docker Compose commands.

## Features

Quay acts as a wrapper around Docker Compose, enhancing it with the ability to:

- **Selectively Run Services** - Run only the services you need:
  - Use `--include web db` to start only specific services
  - Use `--exclude redis` to run everything except certain services
  
- **Override Port Mappings** - Change port bindings without modifying your compose file:
  - Use `--port web:8080:80` to publish a container's port 80 to host port 8080
  - Apply multiple port overrides in a single command
  
- **Retain Docker Compose Functionality** - Quay passes through all standard Docker Compose commands and options
  - Works with all Docker Compose commands (`up`, `down`, `logs`, etc.)
  - Supports Docker Compose flags like `-d` (detached mode)
  - Specify custom compose files with `-f`

Think of Quay as Docker Compose with additional filtering capabilities - perfect for complex applications where you only need to work with specific parts of the stack.

## Installation

### Option 1: Homebrew (macOS and Linux)

```bash
brew tap yarlson/quay
brew install quay
```

### Option 2: Download from Releases

1. Visit the [GitHub Releases page](https://github.com/yarlson/quay/releases)
2. Download the appropriate binary for your operating system and architecture
3. Extract the archive and move the binary to a location in your PATH

```bash
# Example for macOS/Linux
chmod +x quay
sudo mv quay /usr/local/bin/
```

### Option 3: Build from Source

To build from source, you need Go installed on your system:

```bash
# Clone the repository
git clone https://github.com/yarlson/quay.git
cd quay

# Build the binary
go build -o quay .

# Optionally, move the binary to a location in your PATH
sudo mv quay /usr/local/bin/
```

## Usage

To use Quay, you can specify the Docker Compose file and the services you want to manage:

```bash
./quay -f path/to/docker-compose.yml up -d --include web
./quay -f path/to/docker-compose.yml up -d --exclude db
```

### Basic Commands

- **Up**: Start services
  ```bash
  ./quay up -d                                # Run all services
  ./quay up -d --include web --include db     # Run only web and db services
  ./quay up -d --exclude web                  # Run all services except web
  ```
- **Down**: Stop services
  ```bash
  ./quay down
  ```

### Advanced Usage

You can also use Quay to run specific services with custom Docker Compose files:

```bash
./quay -f custom.yml up --include redis
./quay -f custom.yml up --exclude postgres
```

You can redefine published ports for services:

```bash
./quay up -d --port web:8080:80              # Map container port 80 to host port 8080 for web service
./quay up -d --include web --port web:3000:80 # Run only web service with custom port mapping
```

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue if you have feedback or suggestions.

## License

Quay is open-sourced software licensed under the [MIT license](LICENSE).
