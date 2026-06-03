# Webarchive Design

This document details the internal design and data models utilized in the `webarchive` tool. 

## Data Model

The primary data structures are found in the `entity` package. They represent the state and outcomes of archiving tasks.

*   **`Page`**: The root entity. 
    *   **Fields**: `ID` (UUID), `URL`, `Description`, `Owner`, `Tags`, `Headers`, `Cookies`, `Created` timestamp, `Formats` (array of requested formats), and `Status`.
    *   **Status Lifecycle**: `StatusNew` -> `StatusProcessing` -> (`StatusDone` | `StatusFailed` | `StatusWithErrors`).
    *   **Metadata**: Contains extracted meta tags from the target URL like Title, Description, and Encoding.
*   **`Result`**: Represents the outcome of an individual format conversion. Contains the resulting `File` (if successful) or an `Error`.
*   **`File`**: A simple structure holding the generated filename and the raw byte content (e.g., the PDF blob or HTML string).

## Concurrency & Asynchronous Processing

The archiving process is inherently slow due to network latency, large file downloads, and heavy processing (like PDF rendering). 

To prevent blocking API requests, the system is designed asynchronously:
1.  **Channel-based Worker**: When a page is requested, it is immediately written to the database with `StatusNew`, and a pointer is passed to a Go channel (`chan *Page`).
2.  **Worker Loop**: The `Worker` struct (in `entity/worker.go`) runs in a background goroutine. It listens to the channel and processes pages sequentially.
3.  **Concurrent Formatters**: Within a single page processing job, if a user requests multiple formats (e.g., `pdf` and `headers`), the `Worker` spawns a goroutine for each requested format. `sync.WaitGroup` is used to wait for all formats to finish before marking the overall page task as completed.

## Intelligent Caching Layer

Because a single page request might spawn multiple processor tasks, hitting the target URL multiple times would be inefficient and could lead to rate-limiting or inconsistent states.

The `entity.Cache` implements a thread-safe byte buffer:
*   The first processor to execute will fetch the network data and stream it into the `Cache`.
*   Subsequent processors can read from this in-memory `Cache` buffer rather than making a new external network call.

## Storage Design (Badger DB)

`webarchive` uses **BadgerDB**, an embeddable, highly performant key-value store.
*   **No external database server is required**, fulfilling the "simple, fast, and easy-to-use" goal for personal usage.
*   Data is stored locally on the disk (default path: `./db`).
*   Both the page metadata (serialized using msgpack) and the raw generated files are saved into Badger. 
*   Because files (like PDFs) can be several megabytes in size, Badger handles them efficiently compared to traditional relational databases.
