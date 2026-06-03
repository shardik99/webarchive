# Open Items & Roadmap

This document outlines planned features, missing functionality, and potential areas for improvement within the `webarchive` project.

## Planned Features (Roadmap)

According to the initial project roadmap, several features are slated for future development:

1.  **Database Flexibility**: The system currently hardcodes BadgerDB. The architecture is flexible enough (via the `ports` and `adapters` interfaces) to support external SQL databases (like PostgreSQL or SQLite) with or without separate blob storage (like S3 or Minio).
2.  **Companion Chrome Extension**: Developing a browser extension that allows users to easily send their currently viewed page to the archive. This extension should leverage the browser's active session to extract and forward the authenticated headers and cookies directly to the webarchive API.

## Recently Completed Features

*   **Markdown Support**: Implemented a format processor that parses the web article's main content cleanly and converts it into standard Markdown.
*   **HTML with Separate Resources**: A new `html` format has been implemented that saves the raw HTML while traversing, downloading, and relinking all external resources into separate files within the archive.
*   **Tags and Categories**: Added full support for tagging and categorizing archives in the backend API, BadgerDB, and UI.
*   **Authentication & Multi-User Support**: Added middleware authentication and `Owner` scoped filtering to ensure archives belong to specific users.
*   **Authenticated Page Archiving**: The `html` and `singlefile` processors now support fetching pages requiring login by forwarding custom headers, cookies, and session tokens submitted via the API.
*   **Modern Web UI**: The basic dashboard was completely replaced with a modern dark-themed dashboard featuring grid and list views.

## Technical Debt & Improvements

1.  **Binary Dependency Management**: The PDF processor tightly couples the system to the external `wkhtmltopdf` binary. The system should gracefully degrade or provide clear runtime errors if the binary is missing, rather than crashing or failing silently.
2.  **Retry Mechanisms**: If a network request fails during the processing stage, the page is marked as Failed. Implementing exponential backoff retries for transient network errors would improve reliability.
3.  **Cache Eviction / Pruning**: The embedded Badger database will grow indefinitely. A feature to prune old or deleted archives from the local disk is needed.
4.  **API Pagination**: The endpoint that lists all stored pages (`GET /api/v1/pages`) currently returns all records. This needs pagination to prevent memory bloat and slow responses as the archive grows.
