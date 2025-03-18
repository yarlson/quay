# Docker Compose Filter

A CLI tool that filters services from a docker-compose.yml file and pipes the result to docker-compose.

## Installation

1. Clone this repository
2. Install dependencies:
   ```
   go mod download
   ```
3. Build the tool:
   ```
   go build -o compose-filter
   ```

## Usage

```
./compose-filter [options] [service1] [service2] ...
```

### Options

- `-f string`: Path to docker-compose.yml file (default "docker-compose.yml")
- `-cmd string`: Docker compose command to run (default "up -d")
- `-print`: Print filtered docker-compose config without executing

### Examples

To start only the `nginx1` and `nginx3` services:

```
./compose-filter nginx1 nginx3
```

To stop only specific services:

```
./compose-filter -cmd "down" nginx1 nginx3
```

To use a different docker-compose file:

```
./compose-filter -f production-compose.yml nginx1 nginx3
```

To only print the filtered configuration without executing:

```
./compose-filter -print nginx1 nginx3
```

## How It Works

1. Parses the docker-compose.yml file
2. Filters out any services not specified in the command line
3. Generates a new docker-compose configuration in memory
4. Either prints the filtered configuration or pipes it to `docker-compose -f -`

## Features

- Filters docker-compose services based on command-line arguments
- Preserves volumes, networks, and other configuration
- Does not modify the original docker-compose.yml file
- Provides warnings for requested services that don't exist
- Supports various docker-compose commands (up, down, restart, etc.) 