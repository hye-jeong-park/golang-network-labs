# golang-network-labs

This repository contains a minimal Go-based system that demonstrates
HTTP and TCP server interaction, container-based service networking,
concurrent request handling, and lightweight persistence using MariaDB.

The project focuses on keeping the code readable while showing
how HTTP, TCP, database, and Docker-based environments communicate
in a real execution flow.

</br>

## Overview

The system consists of three services running via Docker Compose:

- **API Server**
  - Receives HTTP requests (GET / POST)
  - Supports JSON, form-urlencoded, and multipart/form-data inputs
  - Forwards commands to a TCP server
  - Returns execution results as HTTP JSON responses
  - Stores execution logs in MariaDB

- **TCP Server**
  - Listens for TCP connections
  - Receives JSON requests over TCP
  - Executes OS commands inside a Linux container
  - Restricts executable commands via an allowlist
  - Responds with JSON results

- **MariaDB**
  - Stores execution logs
  - Initialized automatically on container startup

</br>

## Project Structure

```

.
├─ api/
│  ├─ main.go        # HTTP API server
│  └─ Dockerfile
├─ tcp/
│  ├─ main.go        # TCP command execution server
│  └─ Dockerfile
├─ docker-compose.yml
├─ .env.example
├─ .gitignore
└─ README.md

```

---

## Architecture

```

Client (curl / browser)
↓ HTTP
API Server (container, :8080)
↓ TCP (JSON, service name resolution)
TCP Server (container, :9000)
↓
Linux OS Command
↓
MariaDB (container, logs persistence)

````

</br>

## Environment Variables (.env)

Sensitive values are loaded from a `.env` file.
This file **must not be committed to Git**.

### 1. Create `.env`

Create a `.env` file in the project root:

```env
MARIADB_ROOT_PASSWORD=rootpass
MARIADB_DATABASE=appdb
MARIADB_USER=appuser
MARIADB_PASSWORD=apppass
````

### 2. `.env.example`

The repository includes `.env.example` for reference:

```env
MARIADB_ROOT_PASSWORD=change_me
MARIADB_DATABASE=appdb
MARIADB_USER=appuser
MARIADB_PASSWORD=change_me
```

### 3. Git Ignore

`.env` must be excluded via `.gitignore`:

```gitignore
.env
.env.*
```

</br>

## How to Run (Docker Compose)

From the project root:

### 1. Build and start services

```bash
docker compose up -d --build
```

### 2. Check running containers

```bash
docker compose ps
```

### 3. View logs

```bash
docker compose logs --tail=100 api
docker compose logs --tail=100 tcp
docker compose logs --tail=100 mariadb
```

### 4. Stop services

```bash
docker compose down
```

> If `.env` or `docker-compose.yml` changes, recreate containers:

```bash
docker compose up -d --build --force-recreate
```

</br>

## API Usage

### 1. GET /run

```bash
curl "http://localhost:8080/run?cmd=uname%20-a"
```

### 2. POST /run (JSON)

```bash
curl -X POST "http://localhost:8080/run" \
  -H "Content-Type: application/json" \
  -d '{"cmd":"uname -a"}'
```

### 3. POST /run (multipart/form-data)

```bash
curl -X POST "http://localhost:8080/run" \
  -F "cmd=uname -a"
```

Expected response:

```json
{
  "ok": true,
  "output": "Linux ..."
}
```

</br>

## HTML Title Parsing

```bash
curl "http://localhost:8080/title?url=https://example.com"
```

Response:

```json
{
  "url": "https://example.com",
  "title": "Example Domain"
}
```

This endpoint performs an HTTP GET and parses the `<title>` element
from the HTML response. It does not interact with the TCP server.

</br>

## Database (MariaDB)

### Logs Table

The API server automatically creates the following table:

```sql
CREATE TABLE logs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  ts DATETIME NOT NULL,
  cmd VARCHAR(255) NOT NULL,
  ok TINYINT NOT NULL
);
```

### Inspect Logs

```bash
docker exec -it golang-network-labs-mariadb-1 \
  mariadb -uappuser -papppass appdb
```

```sql
SELECT * FROM logs ORDER BY id DESC LIMIT 10;
```
</br>

## Expected System Behavior

The system is considered working correctly when:

* GET and POST `/run` return valid JSON responses
* TCP commands execute successfully inside the container
* Only allowlisted commands are executed
* MariaDB logs are created and updated per request
* `/title` works independently of the TCP server
* Services communicate via Docker service names (not localhost)

</br>

## Technologies Used

* Go (`net/http`, `net`, `bufio`)
* goroutines and channels
* JSON (`encoding/json`)
* MariaDB (`database/sql`, `go-sql-driver/mysql`)
* Docker & Docker Compose
* HTML parsing (`golang.org/x/net/html`)

</br>

## Notes

* Command execution is intentionally restricted via allowlist
* This project is intended for learning and demonstration purposes
* Not suitable for direct production use