# LumbungFS

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

## 📂 Storage Architecture

The container exposes two primary directories for persistent storage:

- `/app/bucket`: Stores all raw and encrypted assets sorted by registered origins snake_case domain names.

_Make sure to mount these directories to local volumes to avoid data loss on container recreation._

---

## 📄 License

LumbungFS is released under the [MIT License](LICENSE). Developed and maintained by [Jefri Herdi Triyanto](https://github.com/jefriherditriyanto).
