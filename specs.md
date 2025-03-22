# Developer Specification for the Multi-Docker-Project CLI

## 1. Overview

This CLI tool is designed to run multiple Docker projects locally in parallel while resolving port conflicts, managing service lifecycles, and providing a universal ingress mechanism. It builds upon the original CLI’s Docker Compose manipulation (using the compose-go library) and extends functionality to support multi-repository projects, manual port overrides, integrated ingress routing, and dynamic environment injection—all while following Effective Go principles and a vertical slices architecture.

---

## 2. Requirements

### 2.1 Functional Requirements

- **Multiple Project Management:**
    - Run multiple Docker Compose files from different directories with distinct context folders.
    - Manually override conflicting ports (e.g., 80 and 443) via CLI options.

- **YAML-Based Project Configuration:**
    - A single YAML configuration file per repository defines:
        - **Project Details:**
            - Local project path, Docker Compose file location, and build context folder.
            - Manual port overrides (e.g., mapping a container’s port 80 to host port 8080).
        - **Environment Variables:**
            - Inline variables using the proper substitution syntax: `${VAR:-default}`.
            - Option to load additional variables from an external `.env` file.
        - **Ingress Configuration (Inline):**
            - Optional ingress settings included within each project’s config (hostnames, URL path patterns, SSL settings using mkcert).
        - **Remote Project Provisioning:**
            - Reference external projects with repository URL and branch.
            - CLI allows branch override via a flag.
            - The CLI auto-clones remote repositories if not already present locally (no scheduled updates).

- **Universal Ingress:**
    - Projects may include an ingress section that defines:
        - Hostname and/or URL path patterns.
        - SSL certificate generation settings (via mkcert or a similar tool).
    - Ingress settings are part of the project configuration rather than a separate file.

- **Lifecycle Management:**
    - Provide commands to manage project lifecycles:
        - **Commands:** `start`, `stop`, `reload`, `status`, and `cleanup`
        - Support operations on individual projects and all projects globally.
        - **Fail-Fast Strategy:** Abort the entire operation immediately if any project fails to deploy or execute correctly.

- **Status and Diagnostics:**
    - The status command reports:
        - Container statuses (running or stopped).
        - Active port mappings (including any manual overrides).
        - Ingress route health via simple HTTP checks.
        - SSL certificate status (ensuring certificates are generated and valid).
        - Basic diagnostic information and error logs when applicable.

- **Logging and Debugging:**
    - Configurable logging with multiple verbosity levels (default: info; with options for debug, warn, error).
    - Option to output logs to the console and/or to a file (via a CLI flag).
    - Structured logging with detailed error messages and stack traces in debug mode.

- **Cleanup Command:**
    - A dedicated command to:
        - Stop projects.
        - Remove unused containers, networks, and images.
        - Purge temporary files (e.g., generated SSL certificates).

- **CLI Argument Handling:**
    - Support overriding branch selections for remote projects via CLI arguments.
    - Provide additional flags for logging, environment variable overrides, and manual port mappings.

### 2.2 Non-Functional Requirements

- **Reliability & Fail-Fast:**  
  Immediate abort on errors during multi-project operations with clear error reporting.
- **Extensibility:**  
  YAML configuration and modular design allow future feature additions.
- **Performance:**  
  Ensure operations (e.g., status checks, Docker Compose manipulations) remain responsive.
- **Security:**  
  Secure handling of sensitive data (environment variables, SSL certs) and proper isolation of project contexts.

---

## 3. Architecture and Design

### 3.1 Overall Structure and Vertical Slices

The CLI will follow a vertical slices architecture, where each feature (or "slice") encompasses all layers (CLI command, configuration, business logic, integration with external systems) for that feature. The main slices include:

1. **Configuration & Environment Slice:**
    - **YAML Parser:**
        - Reads and validates the YAML configuration using a library (gopkg.in/yaml.v3).
        - Implements variable substitution with the correct `${VAR:-default}` format.
        - Loads external `.env` files.
    - **Configuration Schema:**
        - Defines project, ingress, environment, and remote provisioning details (see Section 4).

2. **Compose & Docker Integration Slice:**
    - **Docker Compose Manager:**
        - Uses the compose-go library to parse and modify Docker Compose files.
        - Applies manual port mappings and service filters.
        - **Piping Configuration:**
            - For filtered operations, configuration data is piped via stdin directly to the `docker-compose` command (no temporary file generation).

3. **Ingress Slice:**
    - **Ingress Manager:**
        - Integrates with an ingress solution (or configures Nginx/Traefik as needed) to route traffic based on hostnames/paths.
        - Manages SSL certificate generation via mkcert.
        - Performs simple HTTP health checks on ingress endpoints.

4. **Remote Project Provisioning Slice:**
    - **Remote Handler:**
        - Clones external repositories using Git commands or libraries.
        - Merges external project configurations.
        - Allows branch override via a CLI argument.
        - Minimal use of mocks—favoring integration tests with testcontainers for real-world behavior.

5. **Command Processing Slice:**
    - **Cobra CLI Framework:**
        - Use Cobra for command parsing and subcommand organization.
        - Implements commands: `start`, `stop`, `reload`, `status`, and `cleanup`.
    - **Fail-Fast Handling:**
        - Abort multi-project operations if any step fails, providing a clear error message.

6. **Logging and Diagnostics Slice:**
    - **Logging Module:**
        - Use a structured logging library (e.g., logrus or zap) for configurable verbosity.
        - Default output is to console; an optional CLI flag can redirect logs to a file.

### 3.2 Data Flow

1. **Input:**
    - YAML configuration file containing all project details.
    - CLI arguments (using Cobra) including branch overrides, logging options, and manual port mapping overrides.

2. **Processing:**
    - Configuration parser reads and validates the YAML and environment files.
    - For each project:
        - Remote repositories are cloned (if needed) and integrated.
        - Docker Compose files are parsed and filtered via compose-go.
        - Manual port mappings and ingress settings are applied.
        - Ingress configuration triggers mkcert-based SSL certificate generation and health checks.
    - All configuration changes are piped directly to the docker-compose command via stdin.

3. **Output:**
    - The CLI executes Docker Compose commands using the piped configuration.
    - Status command outputs structured diagnostic data.
    - Logs are produced at the configured verbosity level.

---

## 4. YAML Configuration Schema

Below is an example YAML configuration that follows the updated syntax and structure:

```yaml
projects:
  - name: web-project
    path: ./path/to/web-project
    compose_file: docker-compose.yml
    context: ./web-context
    ports:
      - service: nginx
        host: "8080"
        container: "80"
      - service: nginx
        host: "8443"
        container: "443"
    env:
      # Inline environment variables with default substitution using the proper format:
      DATABASE_URL: "${DATABASE_URL:-postgres://user:pass@localhost:5432/db}"
    env_file: .env
    ingress:
      enabled: true
      hostname: "web-project.local"
      paths:
        - "/"
      ssl:
        provider: mkcert
    remote:
      repo: "https://github.com/example/remote-service.git"
      branch: "main"

  - name: api-project
    path: ./path/to/api-project
    compose_file: docker-compose.yaml
    context: ./api-context
    ports:
      - service: api
        host: "3000"
        container: "3000"
    env:
      API_KEY: "${API_KEY:-default_key}"
    # No ingress for this project
```

Each project entry includes:
- **name:** Unique identifier.
- **path:** Local directory path.
- **compose_file:** Path to the Docker Compose file.
- **context:** Docker build context folder.
- **ports:** List of port mappings with manual overrides.
- **env:** Inline environment variables using the `${VAR:-default}` syntax.
- **env_file:** External `.env` file.
- **ingress:** Optional ingress configuration with hostname, URL paths, and SSL settings.
- **remote:** Optional remote project provisioning details (repository URL and branch).

---

## 5. Error Handling Strategies

- **Fail-Fast Mechanism:**
    - During multi-project operations (e.g., `start`), if any project encounters an error (Docker Compose failure, remote repo issues, ingress misconfiguration), the CLI aborts immediately.
    - Detailed error messages and stack traces (in debug mode) are logged and output.

- **Validation and Input Errors:**
    - The configuration parser validates the YAML schema and substitution syntax.
    - Descriptive error messages are provided for missing, malformed, or conflicting configuration options.

- **CLI Argument Conflicts:**
    - Conflicting CLI arguments (e.g., improper branch overrides or invalid port mappings) trigger a usage message and exit with an error.

- **Logging Errors:**
    - All errors are logged using structured logging, with stack traces available in debug mode.
    - Logs can be directed to both console and an optional file for detailed analysis.

---

## 6. Testing Plan

### 6.1 Unit Testing

- **Configuration Parser:**
    - Validate proper YAML parsing, variable substitution (using `${VAR:-default}`), and loading of `.env` files.
    - Use [testify suites](https://github.com/stretchr/testify) to organize tests.

- **Docker Compose Manager:**
    - Unit tests for filtering services, applying manual port mappings, and ensuring that the generated configuration (piped via stdin) is correct.

- **Ingress Manager:**
    - Test SSL certificate generation (mocking mkcert where necessary).
    - Validate simple HTTP health check functions.

- **Remote Repository Handler:**
    - Simulate cloning, branch override functionality, and error handling when repositories are unreachable.
    - Minimal use of mocks; prefer integration tests with real containers (see below).

- **Command Processing:**
    - Unit tests for each Cobra command (`start`, `stop`, `reload`, `status`, `cleanup`).
    - Verify that errors trigger the fail-fast behavior as expected.

### 6.2 Integration Testing

- **End-to-End Workflow:**
    - Use [testcontainers](https://github.com/testcontainers/testcontainers-go) to spin up temporary Docker environments.
    - Create dummy projects with Docker Compose files and verify:
        - Projects start correctly with manual port overrides.
        - Ingress routes are correctly set up and pass HTTP health checks.
        - Remote repository provisioning works when specifying a repo URL and branch.

- **Logging and Error Handling:**
    - Test that errors during multi-project operations trigger immediate abort and log appropriate details.
    - Validate that the status command outputs the expected diagnostic data.

### 6.3 Testing Tools & Principles

- **Testify Suites:**
    - Organize unit tests into suites for configuration, Docker integration, ingress, remote handling, and command processing.

- **Testcontainers:**
    - Use real Docker containers for integration tests to simulate production scenarios without excessive reliance on mocks.

- **Effective Go & Vertical Slices:**
    - Ensure tests are organized along the vertical slices, covering all layers for each feature end-to-end.

---

## 7. Implementation Considerations

- **Language and Frameworks:**
    - **Go Language:** Adhere to Effective Go principles.
    - **Cobra CLI Framework:** For robust command-line parsing and subcommand structure.
    - **compose-go Library:** For Docker Compose file manipulation.
    - **YAML Parsing:** Use gopkg.in/yaml.v3.
    - **Logging:** Use a structured logging library (logrus or zap).
    - **Git Integration:** Utilize Go Git libraries or shell out to Git commands for remote repository handling.

- **Docker Compose Manipulation:**
    - Instead of generating temporary YAML files, pipe the modified YAML configuration directly to the docker-compose command's stdin.

- **Vertical Slices Architecture:**
    - Organize code into discrete vertical slices (e.g., config, compose, ingress, remote, cmd, logging) so that each feature can be developed and tested end-to-end.

- **CI/CD Pipeline:**
    - Integrate unit and integration tests (using testify and testcontainers) into the CI/CD pipeline to ensure continuous quality and early detection of issues.

---

## 8. Developer Handoff and Next Steps

- **Documentation & Code Comments:**
    - Provide detailed inline documentation following Effective Go practices.
    - Include a comprehensive README with examples of YAML configuration and CLI usage.

- **Repository Structure:**
    - Organize code into packages (e.g., `config`, `compose`, `ingress`, `remote`, `cmd`, `logging`).
    - Include sample configurations and Docker Compose files for development and testing.

- **Development Process:**
    - Begin with the configuration parser and CLI command structure using Cobra.
    - Progress with Docker Compose manipulation (piping via stdin) and integrate the ingress management features.
    - Implement remote repository provisioning last, ensuring integration tests cover cross-repository scenarios.

- **Testing:**
    - Write unit tests with testify suites for each vertical slice.
    - Use testcontainers for integration tests, ensuring minimal use of mocks.
    - Set up a CI pipeline to run tests on every commit.
