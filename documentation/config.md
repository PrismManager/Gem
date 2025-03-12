# Configuration Documentation

This document provides an overview of the configuration structure and options available in the `config.go` file. The configuration is divided into two main parts: **Global Configuration** and **Process Configuration**.

## Global Configuration

The global configuration is loaded from a `config.yaml` file and is used to set up the application's environment, including logging, API settings, and cluster mode.

### Configuration Fields

| Field Name       | Type       | Default Value              | Description                                             |
| ---------------- | ---------- | -------------------------- | ------------------------------------------------------- |
| `log_level`      | `string`   | `"info"`                   | Logging level (e.g., `info`, `debug`, `warn`, `error`). |
| `api_port`       | `int`      | `3456`                     | Port on which the API server will listen.               |
| `socket_path`    | `string`   | `"<config_dir>/gem.sock"`  | Path to the Unix socket file used for communication.    |
| `processes_path` | `string`   | `"<config_dir>/processes"` | Directory where process configurations are stored.      |
| `logs_path`      | `string`   | `"<config_dir>/logs"`      | Directory where process logs are stored.                |
| `cluster_mode`   | `bool`     | `false`                    | Whether the application is running in cluster mode.     |
| `cluster_nodes`  | `[]string` | `[]`                       | List of cluster node addresses (used in cluster mode).  |

### Loading Global Configuration

The global configuration is loaded using the `LoadConfig` function, which reads the `config.yaml` file from the specified directory. If the file does not exist, it creates a default configuration file.

#### Example `config.yaml`

```yaml
log_level: "info"
api_port: 3456
socket_path: "/path/to/gem.sock"
processes_path: "/path/to/processes"
logs_path: "/path/to/logs"
cluster_mode: false
cluster_nodes: []
```

## Process Configuration

Process configuration is used to define how individual processes are managed. Each process configuration is stored in a `.gem` file and includes settings such as the command to run, environment variables, and restart policies.

### Configuration Fields

| Field Name      | Type                | Default Value  | Description                                                |
| --------------- | ------------------- | -------------- | ---------------------------------------------------------- |
| `name`          | `string`            | **Required**   | Name of the process.                                       |
| `command`       | `string`            | **Required**   | Command to execute for the process.                        |
| `args`          | `[]string`          | `[]`           | Arguments to pass to the command.                          |
| `working_dir`   | `string`            | `""`           | Working directory for the process.                         |
| `environment`   | `map[string]string` | `{}`           | Environment variables for the process.                     |
| `restart`       | `string`            | `"on-failure"` | Restart policy (`"always"`, `"on-failure"`, `"no"`).       |
| `max_restarts`  | `int`               | `10`           | Maximum number of restarts before giving up.               |
| `restart_delay` | `int`               | `3`            | Delay (in seconds) before restarting the process.          |
| `cluster`       | `ClusterConfig`     | `{}`           | Cluster configuration for the process.                     |
| `log`           | `LogConfig`         | `{}`           | Logging configuration for the process.                     |
| `auto_start`    | `bool`              | `false`        | Whether the process should start automatically.            |
| `user`          | `string`            | `""`           | User under which the process should run.                   |
| `group`         | `string`            | `""`           | Group under which the process should run.                  |
| `scripts`       | `ScriptsConfig`     | `{}`           | Scripts to run before/after starting/stopping the process. |

### Cluster Configuration

| Field Name  | Type     | Default Value | Description                                        |
| ----------- | -------- | ------------- | -------------------------------------------------- |
| `instances` | `int`    | `0`           | Number of instances to run (used in cluster mode). |
| `mode`      | `string` | `""`          | Cluster mode (`"fork"` or `"cluster"`).            |

### Log Configuration

| Field Name  | Type     | Default Value | Description                                                          |
| ----------- | -------- | ------------- | -------------------------------------------------------------------- |
| `stdout`    | `string` | `""`          | File path for stdout logs.                                           |
| `stderr`    | `string` | `""`          | File path for stderr logs.                                           |
| `rotate`    | `bool`   | `false`       | Whether to rotate logs.                                              |
| `max_size`  | `string` | `""`          | Maximum size of log files before rotation (e.g., `"10MB"`, `"1GB"`). |
| `max_files` | `int`    | `0`           | Maximum number of log files to keep.                                 |

### Scripts Configuration

| Field Name   | Type     | Default Value | Description                                |
| ------------ | -------- | ------------- | ------------------------------------------ |
| `pre_start`  | `string` | `""`          | Script to run before starting the process. |
| `post_start` | `string` | `""`          | Script to run after starting the process.  |
| `pre_stop`   | `string` | `""`          | Script to run before stopping the process. |
| `post_stop`  | `string` | `""`          | Script to run after stopping the process.  |

### Loading Process Configuration

Process configurations are loaded using the `LoadProcessConfig` function, which reads a `.gem` file and returns a `ProcessConfig` object.

#### Example `.gem` File

```yaml
name: "my-process"
command: "python3"
args: ["app.py"]
working_dir: "/path/to/app"
environment:
  ENV: "production"
restart: "on-failure"
max_restarts: 5
restart_delay: 5
cluster:
  instances: 3
  mode: "fork"
log:
  stdout: "/path/to/logs/my-process-stdout.log"
  stderr: "/path/to/logs/my-process-stderr.log"
  rotate: true
  max_size: "10MB"
  max_files: 5
auto_start: true
user: "app-user"
group: "app-group"
scripts:
  pre_start: "/path/to/scripts/pre_start.sh"
  post_start: "/path/to/scripts/post_start.sh"
  pre_stop: "/path/to/scripts/pre_stop.sh"
  post_stop: "/path/to/scripts/post_stop.sh"
```

Both configurations are loaded from YAML files (`config.yaml` for global configuration and `.gem` files for process configuration). Default values are provided for most fields, ensuring that the application can run with minimal configuration.
