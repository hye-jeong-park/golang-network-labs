# golang-network-labs

This repository contains a minimal Go-based system that demonstrates
HTTP and TCP server interaction, JSON-based communication, concurrent
request handling, and lightweight persistence using sqlite3.

The project focuses on keeping the code small and readable while
showing how different layers communicate in a real execution flow.

---

## Overview

The system consists of two servers:

- **API Server**
  - Receives HTTP requests
  - Forwards commands to a TCP server
  - Returns execution results as HTTP JSON responses
  - Stores execution logs in sqlite3

- **TCP Server**
  - Listens for TCP connections
  - Receives JSON requests
  - Executes Windows OS commands
  - Responds with JSON results

---

## Project Structure

```

.
├─ api/
│  └─ main.go        # HTTP API server
├─ tcp/
│  └─ main.go        # TCP command execution server
└─ README.md

```

---

## Architecture

```

Client (curl / browser)
↓ HTTP
API Server (localhost:8080)
↓ TCP (JSON, one-line)
TCP Server (localhost:9000)
↓
Windows OS Command

````

---

## How to Run

### 1. Start the TCP Server

The TCP server executes Windows commands and **must run on Windows**
(not WSL or Docker).

```bash
cd tcp
go run main.go
````

* Listens on TCP port `9000`
* Waits for incoming JSON requests

</br>

### 2. Start the API Server

Open a separate terminal:

```bash
cd api
go run main.go
```

Expected output:

```
api :8080
```

* The API server listens on HTTP port `8080`

---

## How to Test

Run test commands in a **different terminal** from the API server.

### Execute a Windows Command

```powershell
curl.exe "http://localhost:8080/run?cmd=ipconfig"
curl.exe "http://localhost:8080/run?cmd=tasklist"
```

Expected successful response:

```json
{
  "ok": true,
  "output": "Windows IP Configuration ..."
}
```

Criteria for success:

* `"ok"` is `true`
* Command output is present
* Response is valid JSON

</br>

### HTML Title Parsing

```powershell
curl.exe "http://localhost:8080/title?url=https://example.com"
```

Expected response:

```json
{
  "url": "https://example.com",
  "title": "Example Domain"
}
```

This endpoint performs an HTTP GET and parses the `<title>` element
from the HTML response. It does not interact with the TCP server.

---

## sqlite3 Setup and Behavior

### Dependency

The API server uses sqlite3 via the Go driver:

```bash
go get github.com/mattn/go-sqlite3
```

No external database service is required.

</br>

### Database Initialization

When the API server starts:

* A file named `logs.db` is created in the `api/` directory
* A table named `logs` is created automatically

Table schema:

```sql
CREATE TABLE logs (
  ts TEXT,
  cmd TEXT,
  ok INTEGER
);
```

No manual setup is needed.

</br>

### Verifying Database Creation

From the `api/` directory:

```bash
ls
```

Expected file:

```
logs.db
```

</br>

### Inspecting Logs (Optional)

If sqlite3 CLI is available:

```bash
sqlite3 logs.db
```

```sql
SELECT * FROM logs;
```

Example output:

```
2026-02-04T00:12:52Z|ipconfig|1
2026-02-04T00:19:49Z|tasklist|1
```

Column meanings:

* `ts`  : execution timestamp (UTC)
* `cmd` : executed command
* `ok`  : 1 = success, 0 = failure

---

## Expected System Behavior

The system is considered working correctly when:

* `/run?cmd=ipconfig` returns `"ok": true`
* `/run?cmd=tasklist` returns a process list
* Stopping the TCP server does not crash the API server
* `logs.db` is created and updated per request
* `/title` works independently of the TCP server

---

## Technologies Used

* Go (`net/http`, `net`, `bufio`)
* goroutines and channels
* JSON (`encoding/json`)
* sqlite3 (`database/sql`, `github.com/mattn/go-sqlite3`)
* HTML parsing (`golang.org/x/net/html`)

---

## Notes

* The TCP server executes OS commands and should be treated carefully
* This structure is suitable for understanding layered server interaction,
  not for direct production use
