# API Documentation

## Overview

This API server provides endpoints for managing processes, clusters, and retrieving system information. It is built using the Gin framework and supports WebSocket connections for shell access.

## Table of Contents

- [Overview](#overview)
- [Base URL](#base-url)
- [Endpoints](#endpoints)
  - [Process Management](#process-management)
    - [List Processes](#list-processes)
    - [Start a Process](#start-a-process)
    - [Get Process Information](#get-process-information)
    - [Stop a Process](#stop-a-process)
    - [Restart a Process](#restart-a-process)
    - [Get Process Logs](#get-process-logs)
    - [Shell Access via WebSocket](#shell-access-via-websocket)
  - [Cluster Management](#cluster-management)
    - [List Clusters](#list-clusters)
    - [Get Cluster Information](#get-cluster-information)
  - [System Information](#system-information)
    - [Get System Information](#get-system-information)
  - [Health Check](#health-check)
    - [Health Check](#health-check)
- [Helper Functions](#helper-functions)
  - [Logger Middleware](#logger-middleware)
  - [isClusterWorker](#isclusterworker)

## Base URL

The base URL for the API is `/api/v1`.

## Endpoints

### Process Management

#### List Processes

- **URL**: `/api/v1/processes`
- **Method**: `GET`
- **Description**: Lists all processes, excluding cluster workers.
- **Response**:
  - Status Code: `200 OK`
  - Body: Array of `ManagedProcess` objects.

#### Start a Process

- **URL**: `/api/v1/processes`
- **Method**: `POST`
- **Description**: Starts a new process with the provided configuration.
- **Request Body**: `ProcessConfig` object.
- **Response**:
  - Status Code: `201 Created`
  - Body: `ManagedProcess` object.
  - Status Code: `400 Bad Request` if the request body is invalid.
  - Status Code: `500 Internal Server Error` if the process fails to start.

#### Get Process Information

- **URL**: `/api/v1/processes/:name`
- **Method**: `GET`
- **Description**: Retrieves information about a specific process.
- **Response**:
  - Status Code: `200 OK`
  - Body: `ManagedProcess` object.
  - Status Code: `404 Not Found` if the process does not exist.

#### Stop a Process

- **URL**: `/api/v1/processes/:name`
- **Method**: `DELETE`
- **Description**: Stops a specific process.
- **Query Parameters**:
  - `force`: Boolean (default: `false`). If `true`, forcefully stops the process.
- **Response**:
  - Status Code: `200 OK`
  - Body: `{"status": "stopped"}`
  - Status Code: `500 Internal Server Error` if the process fails to stop.

#### Restart a Process

- **URL**: `/api/v1/processes/:name/restart`
- **Method**: `POST`
- **Description**: Restarts a specific process.
- **Response**:
  - Status Code: `200 OK`
  - Body: `{"status": "restarting"}`
  - Status Code: `500 Internal Server Error` if the process fails to restart.

#### Get Process Logs

- **URL**: `/api/v1/processes/:name/logs/:stream`
- **Method**: `GET`
- **Description**: Retrieves logs for a specific process.
- **Path Parameters**:
  - `stream`: Log stream (`stdout` or `stderr`).
- **Query Parameters**:
  - `lines`: Number of log lines to retrieve (default: `100`).
- **Response**:
  - Status Code: `200 OK`
  - Body: `{"logs": [log lines]}`
  - Status Code: `400 Bad Request` if the stream is invalid.
  - Status Code: `500 Internal Server Error` if the logs cannot be retrieved.

#### Shell Access via WebSocket

- **URL**: `/api/v1/processes/:name/shell`
- **Method**: `GET`
- **Description**: Establishes a WebSocket connection for shell access to a specific process.
- **Response**:
  - WebSocket connection.
  - Status Code: `500 Internal Server Error` if the WebSocket upgrade fails or the shell cannot be attached.

### Cluster Management

#### List Clusters

- **URL**: `/api/v1/clusters`
- **Method**: `GET`
- **Description**: Lists all clusters.
- **Response**:
  - Status Code: `200 OK`
  - Body: Array of `ManagedProcess` objects representing clusters.

#### Get Cluster Information

- **URL**: `/api/v1/clusters/:name`
- **Method**: `GET`
- **Description**: Retrieves information about a specific cluster.
- **Response**:
  - Status Code: `200 OK`
  - Body: `ManagedProcess` object.
  - Status Code: `404 Not Found` if the cluster does not exist.
  - Status Code: `400 Bad Request` if the process is not a cluster.

### System Information

#### Get System Information

- **URL**: `/api/v1/system`
- **Method**: `GET`
- **Description**: Retrieves system information.
- **Response**:
  - Status Code: `200 OK`
  - Body: `{"version": "1.0.0", "uptime": "unknown"}`

### Health Check

#### Health Check

- **URL**: `/health`
- **Method**: `GET`
- **Description**: Checks the health of the API server.
- **Response**:
  - Status Code: `200 OK`
  - Body: `{"status": "ok"}`

## Helper Functions

### Logger Middleware

- **Description**: Logs incoming requests with details such as method, status code, latency, client IP, and path.

### isClusterWorker

- **Description**: Checks if a process name is a cluster worker by checking if it ends with `-worker-`.
