# Webarchive Usage

`webarchive` is designed to be a lightweight tool for personal or home-network usage to archive web pages in various formats.

## Prerequisites

*   **Golang**: Version 1.19 or higher (if building from source).
*   **wkhtmltopdf**: This binary must be installed on the host system and available in your `$PATH` if you intend to save pages as PDFs.
*   **Docker**: Optional, but recommended for easiest deployment.

## Deployment & Running

### Option 1: Docker Compose (Recommended)

The easiest way to start the service with all dependencies (including `wkhtmltopdf` pre-installed) is using Docker Compose.

```shell
docker compose up -d webarchive
```

### Option 2: Run locally

```shell
go run ./cmd/server/main.go
```
*Note: Ensure `wkhtmltopdf` is installed on your OS.*

## Configuration

Configuration is handled via environment variables.

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_PATH` | Path for database storage files. | `./db` |
| `API_ADDRESS` | Address and port the API binds to. | `0.0.0.0:5001` |
| `UI_ENABLED` | Enable or disable the built-in web UI. | `true` |
| `LOGGING_DEBUG`| Enable verbose debug logging. | `false` |
| `PDF_LANDSCAPE`| Set PDF orientation to landscape. | `false` |
| `PDF_FILENAME` | Default output filename for generated PDFs. | `page.pdf` |
| `AUTH_ENABLED` | Enable authentication for the API and UI. | `false` |
| `AUTH_BASIC_USERNAME`| Username for Basic Auth (if enabled). | |
| `AUTH_BASIC_PASSWORD`| Password for Basic Auth (if enabled). | |
| `AUTH_PROXY_HEADER` | Header to use for proxy-forwarded authentication. | `Remote-User` |

*Prefix `WEBARCHIVE_` can be used to prevent variable collisions (e.g., `WEBARCHIVE_DB_PATH`).*

## Interacting with the API

The built-in UI is accessible at `http://localhost:5001/` by default. You can also interact programmatically via the REST API.

### 1. Archiving a Page

Submit a POST request to add a page to the processing queue. Supported formats are `headers`, `pdf`, `single_file`, and `html`. You can also pass custom headers and cookies to archive pages that require login.

```shell
curl -X POST "http://localhost:5001/api/v1/pages" \
    -H "Content-Type: application/json" \
    -d '{
          "url": "https://example.com",
          "formats": ["pdf", "html", "single_file"],
          "tags": ["research", "tech"],
          "headers": {
            "Authorization": "Bearer some-token"
          },
          "cookies": {
            "session_id": "abc123xyz"
          }
        }'
```

The response will contain a page `id`.

### 2. Checking Status

Use the `id` from the previous step to check the processing status. Because conversion is asynchronous, you may need to poll this endpoint.

```shell
curl -X GET "http://localhost:5001/api/v1/pages/<PAGE_ID>"
```

Once the `status` is `success`, the `results` array will contain `file_id`s for the generated assets.

### 3. Downloading Archived Files

You can retrieve the actual content (PDF binary, HTML string, etc.) using the page ID and the specific file ID.

```shell
curl -X GET "http://localhost:5001/api/v1/pages/<PAGE_ID>/file/<FILE_ID>" -o archive.pdf
```
