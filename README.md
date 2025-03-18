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

To install Quay, clone this repository and build the binary using Go:

```bash
git clone https://github.com/yourusername/quay.git
cd quay
go build -o quay .
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
