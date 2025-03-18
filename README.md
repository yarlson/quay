# Quay

Quay is a CLI tool designed to manage and filter Docker Compose services. It allows users to specify which services to run, using a Docker Compose file, and provides functionality to execute common Docker Compose commands with a focus on simplicity and usability.

## Features

- **Service Filtering**:
  - Include specific services using the `--include` option
  - Exclude specific services using the `--exclude` option
  - Note: `--include` and `--exclude` options cannot be used together in the same command
- **Custom Compose File Support**: Use a custom Docker Compose file with the `-f` option.
- **Command Flexibility**: Supports various Docker Compose commands like `up`, `down`, `restart`, and more.

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

## Contributing

Contributions are welcome! Please feel free to submit a pull request or open an issue if you have feedback or suggestions.

## License

Quay is open-sourced software licensed under the [MIT license](LICENSE).
