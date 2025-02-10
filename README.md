# 🪫 Ethereum fetcher Go

Ethereum fetcher API

## 🤓 Prerequisites

[Docker](https://www.docker.com/) installed on your system.

## ⚙️ Project setup

```bash
go mod download
```

Before running the project please create `.env` or use the example one.

```shell
cp .env.example .env
```

```shell
API_PORT=
ETH_NODE_URL=

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ethereum-fetcher
```

## 📖 Run Postgres DB with Docker

```bash
docker compose up -d
```

## 🚀 MakeFile

Run build make command with tests

```bash
make all
```

Build the application

```bash
make build
```

Run the application

```bash
make run
```

Create DB container

```bash
make docker-run
```

Shutdown DB Container

```bash
make docker-down
```

DB Integrations Test:

```bash
make itest
```

Live reload the application:

```bash
make watch
```

Run the test suite:

```bash
make test
```

Clean up binary from the last build:

```bash
make clean
```
