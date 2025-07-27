# LittleVSX

> 🧩 Self-hosted Visual Studio Code Extension Marketplace for offline or restricted environments.

[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## Overview

**LittleVSX** is a standalone marketplace server for VS Code / VSCodium designed for air-gapped or secure networks.  
It allows extensions to be hosted and served locally, with automatic asset rewriting and seamless integration with compatible editors.

> ## ⚠️ Disclaimer
>
> LittleVSX implements only the **minimal set of features** required for Visual Studio Code / VSCodium to discover, download, and install extensions from a local source.
>
> This is **not a full replacement** for the official Visual Studio Marketplace.  
> Features such as ratings, publisher verification, search indexing, telemetry, and extension auto-updates are **not implemented**.
>
> This project is intended for **internal use only** in **restricted, secure, or offline environments**, where basic extension delivery is sufficient.  
> It is deliberately minimalist and should **not be used as a public marketplace or exposed to the Internet**.

## ✨ Features

- 📦 **Compatible API** for VS Code & VSCodium
- 🔁 **Automatic extension fetching** from Microsoft Marketplace
- 🖼️ **Smart asset rewriting** (README images, CSS, JS → local links)
- 🔐 **REST API with HTTPS and CORS**
- 🗃️ **SQLite-based database with auto-migration**

## 🚀 Getting Started

1. Clone the repository:

```bash
git clone https://github.com/i2Phoenix/LittleVSX
cd littlevsx
```

2. Install dependencies:

```bash
go mod tidy
```

3. Build the project

```bash
go build -o littlevsx main.go
chmod +x littlevsx
```

3. Start the server

```bash
./littlevsx serve
```

## ⚙️ Configuration

Create a `config.yaml` in the root directory:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  https: true
  cert_file: "./certs/domain.chain.pem"
  key_file: "./certs/domain.key.pem"
  base_url: "https://domain:8080"

extensions:
  directory: "./extensions"

assets:
  directory: "./extensions/assets"
  cache_time: 3600

database:
  path: "./littlevsx.db"
  auto_migrate: true
  log_queries: false

logging:
  level: "info"
  format: "json"
```

### 🔍 Configuration Reference

| Section    | Key          | Description                              | Default             |
| ---------- | ------------ | ---------------------------------------- | ------------------- |
| server     | host         | Address to bind                          | 0.0.0.0             |
|            | port         | Port number                              | 8080                |
|            | https        | Enable HTTPS                             | true                |
|            | cert_file    | Path to TLS certificate                  |                     |
|            | key_file     | Path to private key                      |                     |
|            | base_url     | External base URL for clients            |                     |
| database   | path         | SQLite file path                         | ./littlevsx.db      |
|            | auto_migrate | Auto-create tables                       | true                |
|            | log_queries  | Verbose SQL logging                      | false               |
| extensions | directory    | Directory where .vsix files are stored   | ./extensions        |
| assets     | directory    | Folder for downloaded assets             | ./extensions/assets |
|            | cache_time   | Cache time in seconds                    | 3600                |
| logging    | level        | Log verbosity (debug, info, warn, error) | info                |
|            | format       | Log format (json or text)                | json                |

## 🔧 CLI Usage

```bash
# Start the server
littlevsx serve

# Download an extension from Microsoft Marketplace
littlevsx download ms-python.python

# Remove an extension
littlevsx delete ms-python.python
```

## 📥 Downloading Extensions

1. Visit https://marketplace.visualstudio.com/
2. Find the desired extension
3. Copy the ID from the URL (e.g., `ms-python.python`)
4. Run:

```bash
littlevsx download ms-python.python
```

Once downloaded, the extension becomes available via API.

Asset processing:

- Images, CSS, and JS from the README are extracted
- Assets are saved to the local assets directory
- URLs in the README are rewritten to local paths

## ⚙️ Configuring VS Code or VSCodium to Use LittleVSX

To redirect Visual Studio Code or VSCodium to use LittleVSX as its extension marketplace, modify the `product.json` file.

### For VSCodium (recommended)

1. Locate the `product.json` file:

- **Linux**: `/opt/vscodium/resources/app/product.json`
- **Windows**: `C:\Program Files\VSCodium\resources\app\product.json`
- **macOS**:  
  Open the Applications folder, right-click **VSCodium.app**, choose **"Show Package Contents"**, then navigate to:  
  `/Applications/VSCodium.app/Contents/Resources/app/product.json`

2. Update the following fields to point to your local LittleVSX instance:

```json
{
  "extensionsGallery": {
    "serviceUrl": "https://your-littlevsx-server:8080/_apis/public/gallery",
    "itemUrl": "https://your-littlevsx-server:8080/items",
    "extensionUrlTemplate": "https://your-littlevsx-server:8080/_gallery/{publisher}/{name}/latest"
  }
}
```

Replace `https://your-littlevsx-server:8080` with the actual URL defined in your `config.yaml` under `server.base_url`.

3. Restart VSCodium. You will now see extensions listed from your LittleVSX server instead of the default marketplace.

> ⚠️ **Note:** VS Code (official Microsoft build) enforces strict signature checks and will reject custom marketplaces. Use [VSCodium](https://vscodium.com/) or your own VS Code fork to bypass these restrictions.

## 📚 Use Cases

- Internal developer environments
- Air-gapped networks (government, military, industrial)
- Custom curated extension marketplaces

## 📄 License

MIT License © 2024 Shelushkin Alexey
