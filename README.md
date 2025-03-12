# Gem

A lightweight, fast process manager for Linux/Ubuntu systems written in Go. Gem allows you to manage processes, view tasks, access shells, automate scripts, and view logs with ease.

## Features

- Start, stop, and restart processes
- Monitor process status and resource usage
- Interactive shell access to running processes
- Automated scripts for process management
- Comprehensive logging system
- Cluster support for distributed environments
- Configuration via `.gem` files (similar to Docker's configuration)
- API for integration with other applications
- Lightweight and fast performance

## Installation

```bash
# Clone the repository
git clone https://github.com/prismmanager/gem.git
cd gem

# Build the application
go build -o gem

# Install globally (optional)
sudo mv gem /usr/local/bin/
```

Alternatively, you can use the provided scripts:

```bash
# Using the development script
./scripts/dev.sh build
sudo ./scripts/dev.sh install

# Using make
make build
sudo make install
```

## Usage

### Basic Commands

```bash
# Start a new process
gem start <process-name> --cmd="<command>"

# Start a process using a .gem configuration file
gem start -f config.gem

# List all running processes
gem list

# View process details
gem info <process-name>

# Stop a process
gem stop <process-name>

# Restart a process
gem restart <process-name>

# Access process shell
gem shell <process-name>

# View process logs
gem logs <process-name>
```

### Configuration

Gem uses `.gem` files for process configuration. Example:

```yaml
name: my-app
cmd: node app.js
cwd: /path/to/app
env:
  NODE_ENV: production
  PORT: 3000
restart: always
max_restarts: 10
cluster:
  instances: 4
  mode: fork
log:
  stdout: ./logs/out.log
  stderr: ./logs/error.log
  rotate: true
  max_size: 10M
  max_files: 5
```

Check the `examples` directory for more configuration examples.

## API Usage

Gem provides a REST API for integration with other applications:

```bash
# Start the API server
gem api start

# API is available at http://localhost:3456 by default
```

## Development

### Requirements

- Go 1.21 or higher
- golangci-lint (for linting)

### Development Scripts

The project includes development scripts to make common tasks easier:

```bash
# Build the binary
./scripts/dev.sh build

# Run tests
./scripts/dev.sh test

# Run linter
./scripts/dev.sh lint

# Clean build artifacts
./scripts/dev.sh clean

# Build and run
./scripts/dev.sh run [args]
```

You can also use the Makefile:

```bash
# Build the binary
make build

# Run tests
make test

# Run linter
make lint

# Build for multiple platforms
make release
```

### CI/CD

The project uses GitHub Actions for continuous integration and deployment. The workflow includes:

- Building the application
- Running tests
- Linting the code
- Creating releases for multiple platforms

## License

MIT
