<div align="center">
  <img src="assets/logo.svg" style="margin-bottom: 20px;" alt="Log Bull Logo" width="250"/>

  <h3>Simple log collection and view system</h3>
  <p>From developer to developer. Install from one command, zero config and much easier than ELK or Loki</p>
  
  <!-- Badges -->
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](#)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](#)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](#)
  
  <p>
    <a href="#-features">Features</a> •
    <a href="#-system-requirements">System Requirements</a> •
    <a href="#-installation">Installation</a> •
    <a href="#-usage">Usage</a> •
    <a href="#-license">License</a> •
    <a href="#-contributing">Contributing</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://logbull.com" target="_blank"><strong>🌐 Log Bull website</strong></a>
  </p>
  
  <img src="assets/dashboard.png" alt="Log Bull Dashboard" width="800"/>

</div>

---

## ✨ Features

### 🐳 **Easy Deployment**

- **Docker-based**: Launch in Docker with one command
- **Zero configuration**: Works out of the box
- **Self-hosted**: All your data stays on your infrastructure
- **Open source**: Apache 2.0 licensed

### 📝 **Multi-Language Log Collection**

- **Go**: Native support with structured logging
- **Python**: Compatible with standard logging libraries
- **Java**: Support for popular Java logging frameworks (Log4j, Logback, SLF4J)
- **Node.js**: Winston, Bunyan, and other JavaScript logging libraries
- **C#/.NET**: Serilog, NLog, and Microsoft.Extensions.Logging
- **PHP**: Monolog and PSR-3 compatible loggers
- **Ruby**: Standard Logger, Lograge, and Rails logging
- **And many more**: Supports any application that can send HTTP requests

### 🎯 **Project Management**

- **Multiple projects**: Organize logs by different applications or services
- **Project isolation**: Keep logs separated and organized
- **Easy switching**: Quick project selection in the dashboard

### 👥 **Multi-User Support**

- **Team collaboration**: Add multiple users to your Log Bull instance
- **User management**: Control access and permissions
- **Secure authentication**: Built-in user authentication system

### 📊 **Audit Logging**

- **Complete audit trail**: See who changed what and when
- **User activity tracking**: Monitor all user actions
- **Change history**: Track modifications to projects and settings

### 🔍 **Powerful Log Querying**

- **Advanced search**: Find logs quickly with powerful search capabilities
- **Real-time viewing**: Stream logs as they arrive
- **Filtering**: Filter logs by various criteria
- **Time-based queries**: Search logs within specific time ranges

---

## 💻 System Requirements

Log Bull requires the following minimum system resources to run properly:

- **CPU**: At least 2 CPU cores
- **RAM**: Minimum 4 GB RAM
- **Storage**: 20 GB available disk space (more recommended for log retention)
- **Docker**: Docker Engine 20.10+ and Docker Compose v2.0+
- **Network**: Internet connection for initial setup and updates

---

## 📦 Installation

You have three ways to install Log Bull: automated script (recommended), simple Docker run, or Docker Compose setup.

### Option 1: Automated Installation Script (Recommended, Linux only)

The installation script will:

- ✅ Install Docker with Docker Compose (if not already installed)
- ✅ Set up Log Bull
- ✅ Configure automatic startup on system reboot

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/logbull/logbull/main/install-logbull.sh \
| sudo bash
```

### Option 2: Simple Docker Run

The easiest way to run Log Bull:

```bash
docker run -d \
  --name logbull \
  -p 8080:8080 \
  -v ./logbull-data:/app/data \
  --restart unless-stopped \
  logbull/logbull:latest
```

This single command will:

- ✅ Start Log Bull
- ✅ Store all data in `./logbull-data` directory
- ✅ Automatically restart on system reboot

### Option 3: Docker Compose Setup

Create a `docker-compose.yml` file with the following configuration:

```yaml
version: "3"

services:
  logbull:
    container_name: logbull
    image: logbull/logbull:latest
    ports:
      - "8080:8080"
    volumes:
      - ./logbull-data:/app/data
    restart: unless-stopped
```

Then run:

```bash
docker compose up -d
```

---

## 🚀 Usage

1. **Launch and access Log Bull**: Start Log Bull and navigate to `http://localhost:4005`
2. **Create your first project**: Click "New Project" and copy the generated project ID
3. **Integrate with your application**: Connect Log Bull to your existing logger in your codebase (Python, Go, Node.js, etc.)
4. **Start viewing logs**: Watch your logs stream in real-time in the Log Bull dashboard!
5. **Manage your setup**: Add team members, adjust limits, and configure settings if needed

### 🔑 Resetting Admin Password

If you need to reset the admin password, you can use the built-in password reset command:

```bash
docker exec -it logbull ./main --new-password="YourNewSecurePassword123" --email="admin@example.com"
```

Replace `admin@example.com` with the actual email address of the user whose password you want to reset.

---

## 📝 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

---

## 🤝 Contributing

Contributions are welcome! If you want to contribute to Log Bull but don't know what and how - message the developer on Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)
