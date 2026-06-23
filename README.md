![LumbungFS Banner](./assets/LumbungFS-banner.png)

[![Docker Pulls](https://img.shields.io/docker/pulls/jefriherditriyanto/lumbung-fs?style=flat-square&logo=docker)](https://hub.docker.com/r/jefriherditriyanto/lumbung-fs)
[![Docker Image Size](https://img.shields.io/docker/image-size/jefriherditriyanto/lumbung-fs/latest?style=flat-square&logo=docker)](https://hub.docker.com/r/jefriherditriyanto/lumbung-fs)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jefriherditriyanto/lumbung-fs?style=flat-square&logo=go)](https://go.dev/)

**LumbungFS** is a premium, developer-focused, self-hosted object storage and file-serving platform. It is designed to act as an intelligent gateway for your files, offering dynamic, on-the-fly file compression and encryption, granular routing rules, origin validation, and a beautiful integrated admin dashboard.

---

## 🌟 Key Features

- ⚡ **High-Performance Serving & Uploads**: Serve and upload assets dynamically with high throughput.
- 🔒 **AES-256-GCM Encryption**: Secure sensitive files on-the-fly. Key derivation is securely tied to your system machine ID and optional custom rules keys.
- 🗜️ **Zstandard (zstd) Compression**: Stateless block compression with custom levels (1 to 22) to optimize disk space without compromising speed.
- 🛡️ **Granular Path-Based Rules**: Restrict files by size limits, enforce allowed extensions list, or hook external API endpoints/JWT validation for file access.
- 🌐 **Origin Access Control**: Authorize specific domains to load assets. Block unknown origins instantly and view unauthorized requests logs from the dashboard.
- 🔑 **Temporary Presigned Tokens**: Prepare backend uploads with secure, 1-minute single-use presigned URL tokens.
- 💻 **Premium Admin Dashboard**: Sleek and responsive React-based admin control center for configuration.

---

## 🚀 Quick Start

Run the container using the official image on Docker Hub:

```bash
docker run -d \
  --name lumbung-fs \
  -p 8080:8080 \
  -v $(pwd)/bucket:/app/bucket \
  -e WEB_DASHBOARD_ORIGIN="http://localhost:5173" \
  -e USERNAME="admin" \
  -e PASSWORD="your-secure-password" \
  jefriherditriyanto/lumbung-fs:latest
```

---

## 🐳 Docker Compose Configuration

Create a `docker-compose.yml` file for simple orchestrations:

```yaml
version: "3.8"

services:
  lumbung-fs:
    image: jefriherditriyanto/lumbung-fs:latest
    container_name: lumbung-fs
    ports:
      - "8080:8080"
    volumes:
      - ./bucket:/app/bucket
    environment:
      - WEB_DASHBOARD_ORIGIN=http://localhost:5173
      - USERNAME=admin
      - PASSWORD=your-secure-password
    restart: unless-stopped
```

---

## ⚙️ Environment Variables

| Variable               | Required | Default  | Description                                                                                                                                                   |
| :--------------------- | :------: | :------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `WEB_DASHBOARD_ORIGIN` | **Yes**  |    —     | The exact URL of the dashboard frontend (e.g. `http://localhost:5173`). Standard API requests are strictly blocked if they do not originate from this domain. |
| `USERNAME`             |    No    | `admin`  | Custom admin username. Replaces the default admin credentials.                                                                                                |
| `PASSWORD`             |    No    | `123456` | Custom admin password. Replaces default admin credentials.                                                                                                    |

---

## 📤 Backend File Uploads (via `curl`)

LumbungFS supports two ways of uploading files programmatically. Note that all backend requests must include a `Host` or `Origin` header that matches the registered origin domain:

### 1. Direct Upload using Origin API Key

Perform a direct multipart `POST` to the `/upload` endpoint, passing the target subpath as the `path` form field and the API Key in the `X-API-Key` header:

```bash
curl -X POST http://localhost:8080/upload \
  -H "X-API-Key: YOUR_ORIGIN_API_KEY" \
  -F "path=file" \
  -F "file=@/path/to/local/image.png"
```

Response:

```json
{
  "message": "File uploaded successfully",
  "original_filename": "image.png",
  "filename": "018f7c5e-88cc-75b2-baee-191b7d598583.png",
  "url": "https://yourdomain.com/file/images/018f7c5e-88cc-75b2-baee-191b7d598583.png",
  "path": "yourdomain_com/images/018f7c5e-88cc-75b2-baee-191b7d598583.png",
  "size": 1024
}
```

### 2. Two-Step Presigned Token Upload Flow

Use this flow when you want your frontend clients to upload files directly to storage without exposing the primary API key:

#### **Step 1: Request a Presigned Upload URL (Backend-to-Backend)**

Generate a single-use token valid for 1 minute:

```bash
curl -X POST http://localhost:8080/upload/prepare \
  -H "X-API-Key: YOUR_ORIGIN_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"path": "url/to/file"}'
```

Response:

```json
{
  "presigned_url": "https://yourdomain.com/upload?token=01904a8b-cb62-7e00-a54f-124b893a7493",
  "token": "01904a8b-cb62-7e00-a54f-124b893a7493",
  "path": "images",
  "expires_at": "2026-06-23T09:15:00Z"
}
```

#### **Step 2: Upload File (Client-to-Storage)**

Perform the actual upload using the returned `presigned_url`. No API Key or Host headers are required for the tokenized upload:

```bash
curl -X POST "http://localhost:8080/upload?token=01904a8b-cb62-7e00-a54f-124b893a7493" \
  -F "file=@/path/to/local/image.png"
```

---

## 📂 Storage Architecture

The container exposes two primary directories for persistent storage:

- `/app/bucket`: Stores all raw and encrypted assets sorted by registered origins snake_case domain names.

_Make sure to mount these directories to local volumes to avoid data loss on container recreation._

---

## 📄 License

LumbungFS is released under the [MIT License](LICENSE). Developed and maintained by [Jefri Herdi Triyanto](https://github.com/jefriherditriyanto).
