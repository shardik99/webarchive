# Webarchive Architecture

The `webarchive` project is built using Go and designed following the principles of Hexagonal Architecture (also known as Ports and Adapters). This pattern strictly separates the core business logic from external concerns (like databases, UI, and external binaries), which keeps the core application highly testable and maintainable.

## Overall Structure

The codebase is organized into several key layers:

### 1. `application`
This package serves as the dependency injection and orchestration layer. 
- It wires together all components of the system: the logger, the database, processors, HTTP server, and background worker.
- It is instantiated via `cmd/service/main.go` and manages the start and graceful shutdown lifecycles of all active components (e.g., stopping the REST server and syncing the database to disk).

### 2. `entity`
This is the core business logic layer. It defines what a web archive is and how processing flows:
- **`Page`**: Represents a web page archival request, including URL, requested formats, status (New, Processing, Done, Failed), and associated processed files.
- **`Worker`**: A background job runner that reads an internal Go channel for incoming `Page` jobs and routes them to the appropriate processor. It enables asynchronous processing.
- **`Cache`**: A memory buffer utilized by processors to download the target URL once, allowing multiple format processors (like headers, pdf) to use the same stream without making redundant HTTP requests to the target server.

### 3. `ports/rest`
This layer handles incoming HTTP requests and user interaction:
- **Authentication Middleware**: Intercepts incoming requests, validates tokens/credentials, and populates the user context (`Owner`) for scoping archives.
- **OpenAPI / Ogen**: The API server is generated using `ogen` from an OpenAPI specification (`api/openapi.yaml`). The REST handlers implement the generated interfaces, parse incoming requests into entity structures, and send them to the `Worker`.
- **UI**: Also served via the HTTP port, providing a modern dark-themed web dashboard with grid and list views (found in `ui/`).

### 4. `adapters`
This layer provides concrete implementations for the application's external dependencies:
- **`repository`**: Implementations of storage interfaces. Currently, it utilizes [BadgerDB](https://dgraph.io/docs/badger/) (`badger.go`), a fast key-value store, to persist both the page metadata and the generated files as byte slices.
- **`processors`**: Implementations for downloading and converting web pages into desired formats.
    - `pdf.go`: Relies on `wkhtmltopdf` binary via the `go-wkhtmltopdf` wrapper to generate PDF documents.
    - `headers.go`: Executes a simple HTTP HEAD or GET request and saves the headers.
    - `singlefile.go`: Captures the HTML and its resources (CSS, JS, Images) into a single standalone HTML document.
    - `html.go`: Captures the raw HTML structure of the page without localizing external resources.

## Data Flow Diagram

1. **Request**: The user issues a POST request to `/api/v1/pages` with a URL and desired formats.
2. **Handling**: The `ports/rest` layer validates the request, creates a new `Page` entity in the `repository`, and sends it to the `Worker`'s channel.
3. **Processing**: The `Worker` picks up the page and marks its status as Processing.
4. **Caching & Formatting**: The worker invokes the specific `adapters/processors`. The `Cache` is populated, and processors run concurrently to produce the required files.
5. **Storage**: The generated files (PDF, HTML, etc.) are saved back into the `repository`.
6. **Result**: The page status is updated to Done or Failed, and the user can retrieve the stored files via GET requests.
