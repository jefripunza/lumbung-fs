# LumbungFS - Architecture & Implementation Plan

LumbungFS is a self-hosted, lightweight, and high-performance multi-tenant/multi-domain object storage server. It features a Golang backend using the standard `net/http` package with JWT-based authentication, SQLite3 database managed via GORM, and a modern, premium Vue 3 + Tailwind CSS Web UI Dashboard.

---

## 1. System Request Flow

Below is the request lifecycle for serving and validating requests in LumbungFS:

```mermaid
sequenceDiagram
    autonumber
    actor Client as Client / Browser
    participant Server as core/server.go
    participant CORS as cors.middleware.go
    participant Auth as auth.middleware.go
    participant DB as SQLite3 (GORM)
    participant Disk as Local Disk (bucket/)

    Client->>Server: Request GET/POST /file/...
    Server->>CORS: Process Origin Check
    alt Header Origin/Host NOT in DB
        CORS->>DB: Log to Unknown Origin Table
        CORS-->>Client: Reject with CORS/Forbidden Error
    else Header Origin in DB (is_blocked == true)
        CORS-->>Client: Reject with Forbidden Error (Blocked)
    end
    
    CORS->>Auth: Process Rule Validation
    Note over Auth: Check path rules (e.g. /file/ktp)
    alt Rule Exists
        alt is_max_size == true AND exceeds max size
            Auth-->>Client: Reject (File Size Exceeded)
        else is_extensions == true AND extension mismatch
            Auth-->>Client: Reject (Invalid File Extension)
        else validate_method is set
            Auth->>Auth: Perform validation checks (JWT, external validate_url)
            alt Auth fails
                Auth-->>Client: Reject (Unauthorized / Fallback Redirect)
            end
        end
    end

    Auth->>Disk: Fetch/Write file under bucket/<origin_snakecase>/...
    Disk-->>Server: Return File Data
    Server-->>Client: 200 OK / Content Data
```

---

## 2. File & Folder Directory Structure

```text
lumbung-fs/
├── main.go                                         # Main entry point (only calls core.Start())
├── go.mod                                          # Go dependencies
├── PLAN.md                                         # High-level architecture and implementation phases
├── bucket/                                         # Physical file store & Database directory
│   ├── data.db                                     # SQLite3 database (GORM)
│   ├── password.txt                                # User credentials (MD5 hash of admin:admin)
│   └── <origin_snakecase>/                         # Directories grouped by origins (recursive structure)
├── core/
│   ├── server.go                                   # Main server initialization (database, routers, server start)
│   ├── database/
│   │   └── gorm.database.go                        # GORM database connection (Connect() function)
│   ├── variables/
│   │   ├── path.variable.go                        # System-wide variables and path constants
│   │   └── crypt.go                                # Cryptographic and compression utility helpers
│   ├── middleware/
│   │   ├── cors.middleware.go                      # Origin verification and CORS handling
│   │   └── auth.middleware.go                      # Rule checks and authorization flow
│   └── modules/
│       ├── routes.go                               # Registry for routing modules
│       ├── origin/
│       │   ├── model/
│       │   │   ├── origin.model.go                 # Origin model structure (GORM)
│       │   │   └── unknown_origin.model.go         # UnknownOrigin model structure (GORM)
│       │   ├── origin.handler.go                   # Handlers for CRUD Origin
│       │   └── origin.router.go                    # Routers for Origin module
│       ├── rule/
│       │   ├── model/
│       │   │   ├── rule.model.go                   # Rule model structure (GORM)
│       │   │   └── rule_file.model.go              # Association model for rules (if needed)
│       │   ├── rule.handle.go                      # Handlers for CRUD Rules & Rule checks
│       │   └── rule.router.go                      # Routers for Rule module
│       ├── file-explorer/
│       │   ├── file_explorer.handler.go            # File uploading, downloading, listing, and folder creation
│       │   └── file-explorer.router.go             # Routers for browsing/uploading/downloading
│       └── upload/
│           ├── upload.handler.go                   # Upload handler and uploader HTML page template
│           └── upload.router.go                    # Routers for upload module (/upload, /upload/prepare)
└── web/                                            # Frontend Vue 3 + Tailwind CSS source
```

---

## 3. System Architecture & Core Database Schema

LumbungFS is designed to serve files for multiple domains (origins) from a single backend service, with custom validation rules and traffic analysis.

### SQLite Database (`./bucket/data.db`)

We will use GORM to manage the following tables:

#### **Origin Table** (Allowed domains)

- `id` (UUIDv7, Primary Key)
- `domain` (string, Unique)
- `is_blocked` (boolean)
- `api_key` (string, API Key for backend upload authentication)

#### **Unknown Origin Table** (Logged traffic for unregistered domains)

- `id` (UUIDv7, Primary Key)
- `domain` (string)
- `access_at` (datetime)
- `ip_address` (string)

#### **Rule Table** (Path-based rules for origins)

- `id` (UUIDv7, Primary Key)
- `origin_id` (UUIDv7, Foreign Key to Origin)
- `path` (string, e.g., "ktp" for "/file/ktp")
- `validate_method` (string, dropdown: JWT, headers, cache)
- `validate_headers` (string, comma-separated list of required header keys for "headers" validation)
- `validate_url` (string, target service validation URL)
- `validate_fallback_url` (string, optional backend fallback template)
- `is_max_size` (boolean)
- `value_max_size` (integer)
- `value_unit_size` (string: KB, MB, GB)
- `is_extensions` (boolean)
- `value_extensions` (string, comma-separated, e.g., "png,jpg,jpeg")
- `is_compress` (boolean)
- `compress_level` (integer, default 3)
- `is_encrypt` (boolean)
- `encryption_key` (string, optional)

---

## 4. Component Implementation Details

### A. Main Entry & Server Bootstrapping
- **`./main.go`**: Minimalist. It only imports `./core` and calls `core.Start()`.
- **`./core/server.go`**:
  - Connects to SQLite3 database via `database.Connect()`.
  - Performs schema auto-migrations for GORM models.
  - Setup routing: UI dashboard endpoints and file serving routes.
  - Checks for `./bucket/password.txt` file (MD5 password checker). If missing, sets up `admin:admin` hash (`e10adc3949ba59abbe56e057f20f883e`).
- **`./core/database/gorm.database.go`**:
  - Configures GORM to connect to sqlite3 (`./bucket/data.db`) without using CGO.

### B. File Operations & Path Handling
- **File Access Pattern**: When a request queries `GET /file/user/ktp/uuid.jpeg` from origin `domain1.com`:
  1. Find origin domain in DB, convert to snake_case (`domain1_com`).
  2. Locate file on disk at `./bucket/domain1_com/user/ktp/uuid.jpeg`.
- **File Upload & Naming**:
  - Rename files to a generated `UUIDv7` string while keeping the original extension (e.g. `018f7c5e-88cc-75b2-baee-191b7d598583.png`).
  - Automatically create nested target directories recursively on disk before saving.

### C. CORS & Rules Middleware
- **Domain Verification Middleware**:
  - Validates the request domain (resolved via `Origin`, `X-Forwarded-Host`, `X-Original-Host`, `Referer`, or `Host` headers) against the registered origins in the database.
  - If not found: logs or updates the `unknown_origins` database logs, then returns a `Forbidden` error.
  - If found but `is_blocked == true`: block access immediately.
- **Rule Verification Middleware**:
  - Analyzes the request path. If a rule exists for this path:
    - Perform request content size checks and file extension verification.
    - If `validate_method` is set, verify via `validate_url`.
  - For uploads (POST/PUT), if no matching path rule is found, the upload is blocked by default (`403 Forbidden`).
- **REST API Upload**:
  - Exposes `POST /upload` endpoint for backend clients.
  - Authenticated via the origin's generated API Key (`X-API-Key` or `Authorization: Bearer <key>`).
  - Validates matching Host or Origin headers and enforces matching path rules.
- **Cascade Deletion**:
  - When deleting an origin, a GORM database transaction explicitly cascade deletes all associated rules.

### D. Frontend Management Dashboard
- Built with **Vue 3** + **Tailwind CSS**.
- **Features**:
  - Premium look (dark mode base, sleek card components, beautiful layout).
  - Admin login validation against password file (JWT authenticated session).
  - Tables to view allowed Origins and logs of Unknown Origins.
  - Interface to configure custom path rules for each domain.
  - Fully-functional **File Explorer** allowing users to create folders, upload files, browse structures, and download files.

### E. Presigned URL Upload Flow
- **Token Generation**: Exposes `/upload/prepare` for authenticated clients to prepare an upload with a specific target path. Generates a temporary token valid for 1 minute stored in the `presigned_urls` table.
- **Unified Upload Handler (`/upload`)**:
  - `GET /upload?token=TOKEN`: Serves a self-contained, beautiful, responsive HTML uploader page using the Denim/Moss palette.
  - `POST /upload?token=TOKEN`: Performs a public multipart upload using the token.
  - `POST /upload`: Performs API Key authenticated backend uploads.
  - Returns a premium, responsive HTML error page for invalid or expired tokens (unless the request accepts `application/json`, in which case it returns a JSON error response).

### F. File Storage Compression & Encryption
- Rules can have `compress file` (`is_compress`) and `encrypt file` (`is_encrypt`) flags enabled.
- Compression uses Zstandard (zstd) compression via `github.com/klauspost/compress/zstd`. If enabled, a custom compression level from 1 to 22 (default 3) can be selected in the rule's `compress_level` field.
- Encryption uses AES-256-GCM. The encryption/decryption key (32 bytes) is derived as follows:
  1. Retrieve the system's hardware UUID/Machine ID via `"github.com/denisbrodbeck/machineid"`.
  2. Retrieve the optional `encryption_key` set on the matched path rule.
  3. Concatenate them as `"[machine_id]:[encryption_key]"`.
  4. Generate a UUIDv5 of the concatenated string using `uuid.NewSHA1`.
  5. Format the UUID as a hexadecimal string without dashes (exactly 32 characters/bytes) and use it as the AES key.
- On client request/serving (via `/file/...` endpoint) or admin file download:
  - If the matching path rule has compression or encryption enabled, the file content is decrypted and/or decompressed transparently.
  - The MIME type is dynamically detected from the processed byte content using `http.DetectContentType` before serving.
- On rule creation or update:
  - If the rule's encryption state transitions from unchecked to checked (`is_encrypt` becomes true), all existing files under the rule's path are recursively encrypted on disk using the derived key.
  - If it transitions from checked to unchecked (`is_encrypt` becomes false), all existing files under the rule's path are recursively decrypted back to their original unencrypted state.
  - If the encryption key is modified while encryption remains enabled, all files under the path are decrypted with the old key and re-encrypted with the new key.
  - If the path is renamed, files under the old path are decrypted (if it was encrypted) and files under the new path are encrypted (if the new rule specifies encryption).


