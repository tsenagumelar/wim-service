# WIM Service

**Weigh In Motion Service** - Service monitoring kendaraan dengan ANPR dan AXLE Weight Sensor.

---

## 📑 Table of Contents

### 🎯 Getting Started
- [Tentang Aplikasi](#tentang-aplikasi)
- [Fitur Utama](#fitur-utama)
- [Quick Start](#quick-start)
- [Configuration](#configuration)

### 🏗️ Architecture & Deployment
- [Modular Service Architecture](#modular-service-architecture)
- [Cara Menjalankan Services](#cara-menjalankan-services)
- [Production Deployment](#production-deployment)
- [Multi-Site Deployment](#multi-site-deployment)

### 🔌 API Reference
- [API Endpoints](#api-endpoints)
- [Authentication](#authentication)
- [API Testing Guide](#api-testing-guide)
- [Upload Image](#upload-image)

### 🚗 Features & Technical Details
- [Vehicle Correlation](#vehicle-correlation)
- [Database Schema](#database-schema)

### 👨‍💻 Development
- [Development Setup](#development-setup)
- [Testing](#testing)
- [Bagian yang Perlu Diperbaiki](#bagian-yang-perlu-diperbaiki)

### 📊 Operations
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Performance Tips](#performance-tips)

---

## Tentang Aplikasi

WIM Service adalah sistem monitoring kendaraan yang menggabungkan data dari berbagai sensor:

- **ANPR (Automatic Number Plate Recognition)** - Mendeteksi plat nomor kendaraan
- **AXLE Sensor** - Mengukur jumlah axle dan dimensi kendaraan
- **Vehicle Correlation** - Menggabungkan data ANPR dan Axle berdasarkan waktu

Sistem ini support **multi-site deployment** dengan database terpusat.

---

## Fitur Utama

### 1. ANPR Processing

- Monitor FTP untuk file XML dan gambar ANPR
- Upload otomatis ke MinIO storage (optional)
- Simpan metadata ke PostgreSQL
- Support multiple camera locations

### 2. AXLE Weight Processing

- Monitor FTP untuk data axle weight
- Process XML dari sensor WIM
- Tanpa data plat nomor (data plat dari ANPR)

### 3. Vehicle Data Correlation

- **Time-based matching** - Gabungkan ANPR + Axle berdasarkan waktu (≤5 detik)
- **Automatic correlation** - Real-time via database trigger
- **Bi-directional** - Works apapun yang masuk duluan (ANPR atau Axle)
- Lihat detail: [Vehicle Correlation](#vehicle-correlation)

### 4. REST API + JWT Authentication

- Token-based auth (expired 24 jam)
- Protected endpoints
- CORS enabled
- File upload ke MinIO

### 5. Multi-Site Architecture

- Deploy di multiple lokasi
- Database PostgreSQL terpusat
- Site configuration per lokasi

---

## Quick Start

### Prerequisites

- Go 1.24+
- Docker + Docker Compose / Portainer
- PostgreSQL database
- MinIO server (optional)
- FTP server untuk ANPR/AXLE data

### Setup dengan Portainer (Recommended)

1. **Build & Push Docker Image**

```bash
git clone https://github.com/tsenagumelar/wim-service.git
cd wim-service

docker build -t tsenagumelar/wim-service:latest .
docker push tsenagumelar/wim-service:latest
```

2. **Deploy di Portainer**

- Login ke Portainer → Stacks → Add Stack
- Copy isi `portainer-stack.yml`
- Edit environment variables (DATABASE_URL, JWT_SECRET, FTP config, dll)
- Deploy the stack

3. **Verifikasi**

```bash
curl http://localhost:4000/health
# Response: {"status":"ok","service":"wim-service"}
```

**Default credentials:**

- Username: `admin`
- Password: `admin123`

### Setup Local Development

1. **Clone & Install**

```bash
git clone https://github.com/tsenagumelar/wim-service.git
cd wim-service
go mod download
```

2. **Configuration**

```bash
cp .env.example .env
# Edit .env dengan konfigurasi Anda
```

3. **Run Services**

WIM Service dipisah menjadi 3 service independen. Jalankan semua sekaligus:

```bash
./start-all.sh
```

Atau jalankan secara manual (buka 3 terminal):

```bash
# Terminal 1 - API Server
go run cmd/api/main.go

# Terminal 2 - ANPR Watcher
go run cmd/anpr-watcher/main.go

# Terminal 3 - AXLE Watcher
go run cmd/axle-watcher/main.go
```

---

## Configuration

### Required Environment Variables

```bash
# Database
DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=disable"

# API Server
API_PORT=4000
JWT_SECRET="min-32-karakter-secret-key"

# Site Configuration
SITE_CODE="SITE001"
SITE_NAME="Lokasi Site 1"

# ANPR FTP
ANPR_FTP_HOST="192.168.1.100:21"
ANPR_FTP_USER="ftpuser"
ANPR_FTP_PASS="ftppass"
ANPR_FTP_DIR="/anpr/"
ANPR_FTP_INTERVAL_SEC=5

# AXLE FTP
AXLE_FTP_HOST="192.168.1.100:21"
AXLE_FTP_USER="ftpuser"
AXLE_FTP_PASS="ftppass"
AXLE_FTP_DIR="/axle/"
AXLE_FTP_INTERVAL_SEC=5

# MinIO Storage (optional - kosongkan jika tidak digunakan)
ANPR_MINIO_ENDPOINT="s3.example.com"
ANPR_MINIO_ACCESS_KEY="admin"
ANPR_MINIO_SECRET_KEY="password"
ANPR_MINIO_BUCKET="anpr"
ANPR_MINIO_USE_SSL=false

AXLE_MINIO_ENDPOINT="s3.example.com"
AXLE_MINIO_ACCESS_KEY="admin"
AXLE_MINIO_SECRET_KEY="password"
AXLE_MINIO_BUCKET="axle"
AXLE_MINIO_USE_SSL=false

ATTACHMENT_MINIO_ENDPOINT="s3.example.com"
ATTACHMENT_MINIO_ACCESS_KEY="admin"
ATTACHMENT_MINIO_SECRET_KEY="password"
ATTACHMENT_MINIO_BUCKET="attachment"
ATTACHMENT_MINIO_USE_SSL=false
```

---

## Modular Service Architecture

Aplikasi WIM Service telah dipisah menjadi **3 service independen** yang dapat berjalan secara terpisah. Ini membuat sistem lebih robust - jika satu service error, service lainnya tetap berjalan.

### 📦 Struktur Service

```
wim-service/
├── cmd/
│   ├── api/              # API Server
│   ├── anpr-watcher/     # ANPR FTP Watcher
│   └── axle-watcher/     # AXLE FTP Watcher
└── start-all.sh          # Helper script
```

### 📊 Detail Setiap Service

#### 1. API Server (`cmd/api/main.go`)

**Fungsi:**
- REST API endpoints
- Authentication (JWT)
- File upload ke MinIO

**Port:** 4000 (default, bisa diubah di `.env`)

**Endpoints:**
- `GET  /health` - Health check
- `POST /api/auth/login` - Login
- `GET  /api/auth/profile` - Get profile (protected)
- `POST /api/attachment/upload` - Upload image (protected)

**Dependencies:**
- Database (PostgreSQL)
- MinIO (bucket: attachment)

---

#### 2. ANPR Watcher (`cmd/anpr-watcher/main.go`)

**Fungsi:**
- Monitor FTP server untuk file ANPR
- Process XML metadata
- Upload images ke MinIO
- Simpan data ke database
- Vehicle dimension detection (optional)

**Dependencies:**
- Database (PostgreSQL)
- FTP Server (ANPR)
- MinIO (bucket: anpr)

**Interval:** 5 detik (default, bisa diubah di `.env`)

---

#### 3. AXLE Watcher (`cmd/axle-watcher/main.go`)

**Fungsi:**
- Monitor FTP server untuk file AXLE
- Process weighing data
- Upload ke MinIO
- Simpan data ke database

**Dependencies:**
- Database (PostgreSQL)
- FTP Server (AXLE)
- MinIO (bucket: axle)

**Interval:** 5 detik (default, bisa diubah di `.env`)

---

## Cara Menjalankan Services

### Opsi 1: Jalankan Semua Service Sekaligus (Recommended)

```bash
./start-all.sh
```

Script ini akan membuka 3 terminal terpisah untuk setiap service.

### Opsi 2: Jalankan Service Secara Manual

Buka 3 terminal terpisah dan jalankan:

**Terminal 1 - API Server:**
```bash
go run cmd/api/main.go
```

**Terminal 2 - ANPR Watcher:**
```bash
go run cmd/anpr-watcher/main.go
```

**Terminal 3 - AXLE Watcher:**
```bash
go run cmd/axle-watcher/main.go
```

### Opsi 3: Jalankan Hanya Service yang Dibutuhkan

Anda bisa menjalankan hanya service tertentu sesuai kebutuhan:

```bash
# Hanya API Server
go run cmd/api/main.go

# Hanya ANPR Watcher
go run cmd/anpr-watcher/main.go

# Hanya AXLE Watcher
go run cmd/axle-watcher/main.go
```

---

## Production Deployment

### 🔧 Build Binary Terpisah

Untuk production, build setiap service menjadi binary terpisah:

```bash
# Build API Server
go build -o bin/wim-api cmd/api/main.go

# Build ANPR Watcher
go build -o bin/wim-anpr-watcher cmd/anpr-watcher/main.go

# Build AXLE Watcher
go build -o bin/wim-axle-watcher cmd/axle-watcher/main.go
```

Lalu jalankan:

```bash
./bin/wim-api &
./bin/wim-anpr-watcher &
./bin/wim-axle-watcher &
```

### Dengan systemd (Linux)

Buat 3 service file:

**`/etc/systemd/system/wim-api.service`:**
```ini
[Unit]
Description=WIM API Server
After=network.target postgresql.service

[Service]
Type=simple
User=wim
WorkingDirectory=/opt/wim-service
ExecStart=/opt/wim-service/bin/wim-api
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**`/etc/systemd/system/wim-anpr-watcher.service`:**
```ini
[Unit]
Description=WIM ANPR FTP Watcher
After=network.target postgresql.service

[Service]
Type=simple
User=wim
WorkingDirectory=/opt/wim-service
ExecStart=/opt/wim-service/bin/wim-anpr-watcher
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

**`/etc/systemd/system/wim-axle-watcher.service`:**
```ini
[Unit]
Description=WIM AXLE FTP Watcher
After=network.target postgresql.service

[Service]
Type=simple
User=wim
WorkingDirectory=/opt/wim-service
ExecStart=/opt/wim-service/bin/wim-axle-watcher
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable dan start services:

```bash
sudo systemctl enable wim-api wim-anpr-watcher wim-axle-watcher
sudo systemctl start wim-api wim-anpr-watcher wim-axle-watcher
sudo systemctl status wim-api wim-anpr-watcher wim-axle-watcher
```

### Dengan Docker Compose

Lihat `docker-compose.yml` untuk deployment dengan Docker.

### ✅ Keuntungan Arsitektur Modular

1. **Reliability** - Jika satu service crash, yang lain tetap jalan
2. **Scalability** - Bisa deploy service di server berbeda
3. **Maintenance** - Bisa restart/update satu service tanpa ganggu yang lain
4. **Monitoring** - Lebih mudah track performance per service
5. **Resource Management** - Bisa alokasikan resource berbeda per service

---

## Multi-Site Deployment

Untuk deployment di multiple site dengan data terpusat:

1. Setup PostgreSQL terpusat (cloud/VPS)
2. Deploy service di setiap site
3. Setiap site punya:
   - `SITE_CODE` unik (contoh: SITE001, SITE002)
   - FTP config lokal
   - Database connection ke server pusat

**Example:**

Site A:

```bash
SITE_CODE=SITE001
SITE_NAME="Jakarta Toll Gate 1"
DATABASE_URL=postgres://user@central-server:5432/wim_db
```

Site B:

```bash
SITE_CODE=SITE002
SITE_NAME="Bandung Toll Gate 2"
DATABASE_URL=postgres://user@central-server:5432/wim_db
```

Query data from all sites:

```sql
SELECT site_code, COUNT(*) FROM view_vehicle_complete
GROUP BY site_code;
```

---

## API Endpoints

### Public Endpoints (No Authentication Required)

| Method | Endpoint             | Description  |
| ------ | -------------------- | ------------ |
| GET    | `/health`            | Health check |
| POST   | `/api/auth/login`    | Login        |

### Protected Endpoints (Require JWT Token)

| Method | Endpoint                   | Description         |
| ------ | -------------------------- | ------------------- |
| GET    | `/api/auth/profile`        | Get user profile    |
| POST   | `/api/attachment/upload`   | Upload image        |

---

## Authentication

### Login

**Request:**

```bash
POST /api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-12-27T10:00:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@wim.local",
      "role": "admin"
    }
  }
}
```

**Alternative - Login with Email:**

```bash
curl -X POST http://localhost:4000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@wim.local",
    "password": "admin123"
  }'
```

**Error Response (Invalid Credentials):**

```json
{
  "success": false,
  "message": "Invalid username or password"
}
```

### Get Profile (Protected)

```bash
GET /api/auth/profile
Authorization: Bearer <token>
```

**Success Response:**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@wim.local",
    "role": "admin",
    "created_at": "2025-12-25T08:00:00Z"
  }
}
```

**Error Response (No Token):**

```json
{
  "success": false,
  "message": "Missing or invalid token"
}
```

**Error Response (Invalid Token):**

```json
{
  "success": false,
  "message": "Invalid or expired token"
}
```

### Token Expiration

JWT tokens expire setelah **24 jam** (default).

Jika token expired, login ulang untuk mendapatkan token baru:

```bash
# Check if token expired
curl -X GET http://localhost:4000/api/auth/profile \
  -H "Authorization: Bearer $TOKEN"

# If expired, login again
TOKEN=$(curl -s -X POST http://localhost:4000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.token')
```

---

## API Testing Guide

### Complete Testing Flow

#### Step 1: Login to Get Token

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:4000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.data.token')

# Verify token saved
echo "Token: $TOKEN"
```

#### Step 2: Test Protected Endpoints

**Get Profile:**
```bash
curl -X GET http://localhost:4000/api/auth/profile \
  -H "Authorization: Bearer $TOKEN"
```

**Upload Image:**
```bash
curl -X POST http://localhost:4000/api/attachment/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "image=@/path/to/image.jpg"
```

### Testing with Postman

#### 1. Login

**Request:**
- Method: `POST`
- URL: `http://localhost:4000/api/auth/login`
- Headers:
  ```
  Content-Type: application/json
  ```
- Body (raw JSON):
  ```json
  {
    "username": "admin",
    "password": "admin123"
  }
  ```

**Save the token** from response untuk digunakan di request selanjutnya.

#### 2. Get Profile

**Request:**
- Method: `GET`
- URL: `http://localhost:4000/api/auth/profile`
- Headers:
  ```
  Authorization: Bearer <paste_your_token_here>
  ```

#### 3. Upload Image

**Request:**
- Method: `POST`
- URL: `http://localhost:4000/api/attachment/upload`
- Headers:
  ```
  Authorization: Bearer <paste_your_token_here>
  ```
- Body (form-data):
  - Key: `image`
  - Type: `File`
  - Value: Select your image file

---

## Upload Image

### Endpoint

```
POST /api/attachment/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

### Request

```bash
curl -X POST http://localhost:4000/api/attachment/upload \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -F "image=@/path/to/your/image.jpg"
```

### Allowed Image Types

- `.jpg`, `.jpeg`
- `.png`
- `.gif`
- `.webp`

### Response

**Success:**
```json
{
  "success": true,
  "file_path": "attachment/550e8400-e29b-41d4-a716-446655440000-image.jpg",
  "message": "File uploaded successfully"
}
```

**Error (No File):**
```json
{
  "success": false,
  "message": "No image file provided"
}
```

**Error (Invalid File Type):**
```json
{
  "success": false,
  "message": "Invalid file type. Allowed: .jpg, .jpeg, .png, .gif, .webp"
}
```

**Error (No Token):**
```json
{
  "success": false,
  "message": "Missing or invalid token"
}
```

### File Storage Structure

Uploaded files disimpan di MinIO dengan struktur:

```
bucket: attachment
path: attachment/uuid-filename.ext

Example:
attachment/550e8400-e29b-41d4-a716-446655440000-profile.jpg
```

---

## Vehicle Correlation

### Problem

- Data ANPR (dengan plat nomor) dan Axle (tanpa plat nomor) masuk terpisah
- Bagaimana menggabungkan tanpa plate matching?

### Solution: Time-Based Matching

Sistem otomatis mencocokan berdasarkan:

1. Same site (`site_id`)
2. Time difference ≤ 5 seconds (configurable)
3. Nearest match (yang paling dekat waktunya)

### How It Works

**Scenario Normal:**

```
10:00:00 → ANPR detects "B 1234 XYZ"
           ↓ Search Axle dalam 5 detik → NOT FOUND
           ↓ Create record: status='ANPR_ONLY'

10:00:03 → Axle measures vehicle (no plate)
           ↓ Search ANPR dalam 5 detik → FOUND (diff=3s)
           ↓ Create record: status='MATCHED'

Result: 1 record dengan plate "B 1234 XYZ" + Axle data
```

### Correlation Status

| Status      | Description                       |
| ----------- | --------------------------------- |
| `MATCHED`   | ANPR + Axle berhasil digabung ✅  |
| `ANPR_ONLY` | ANPR saja, menunggu Axle          |
| `AXLE_ONLY` | Axle saja, plate='UNKNOWN' (rare) |

### Installation

```bash
psql -d wim_db -f migrations/200_vehicle_correlation.sql
```

### Query Examples

**Matched vehicles:**

```sql
SELECT * FROM view_vehicle_complete
WHERE correlation_status = 'MATCHED'
ORDER BY first_detected_at DESC;
```

**Match rate per site:**

```sql
SELECT
    site_code,
    COUNT(*) as total,
    SUM(CASE WHEN correlation_status = 'MATCHED' THEN 1 ELSE 0 END) as matched,
    ROUND(100.0 * SUM(CASE WHEN correlation_status = 'MATCHED' THEN 1 ELSE 0 END) / COUNT(*), 2) as match_rate_pct
FROM view_vehicle_complete
GROUP BY site_code;
```

---

## Database Schema

### Main Tables

**transact_vehicle** - Vehicle correlation result

```sql
- anpr_id (FK → transact_anpr_capture)
- axle_id (FK → transact_axle_capture)
- plate_no (from ANPR only)
- correlation_status (MATCHED/ANPR_ONLY/AXLE_ONLY)
- time_diff_seconds
```

**view_vehicle_complete** - Complete vehicle data view

```sql
SELECT * FROM view_vehicle_complete
-- Auto-JOIN ANPR + Axle + Site data
```

**Site configuration:**

```sql
CREATE TABLE site (
    id uuid PRIMARY KEY,
    code varchar(50) UNIQUE,
    name varchar(200)
);
```

---

## Development Setup

### 1. Prerequisites

Install tools berikut:

```bash
# Go 1.24+
go version

# PostgreSQL
psql --version

# Docker (optional)
docker --version
docker-compose --version

# Git
git --version
```

### 2. Clone Repository

```bash
git clone https://github.com/tsenagumelar/wim-service.git
cd wim-service
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Setup Database

**Option A: PostgreSQL Lokal**

```bash
# Login ke PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE wim_db;

# Create user
CREATE USER wim_user WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE wim_db TO wim_user;

# Exit
\q
```

**Option B: Docker PostgreSQL**

```bash
docker run -d \
  --name wim-postgres \
  -e POSTGRES_DB=wim_db \
  -e POSTGRES_USER=wim_user \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:15-alpine
```

### 5. Run Migrations

```bash
# Run vehicle correlation migration
psql -U wim_user -d wim_db -f migrations/200_vehicle_correlation.sql
```

### 6. Setup MinIO (Optional)

```bash
docker run -d \
  --name wim-minio \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=admin \
  -e MINIO_ROOT_PASSWORD=admin12345 \
  minio/minio server /data --console-address ":9001"

# Access console: http://localhost:9001
# Create buckets: anpr, axle, attachment
```

### 7. Configure Environment

```bash
cp .env.example .env
```

Edit `.env`:

```bash
# Database
DATABASE_URL="postgres://wim_user:password@localhost:5432/wim_db?sslmode=disable"

# API
API_PORT=4000
JWT_SECRET="dev-secret-key-change-in-production-min-32-chars"

# Site
SITE_CODE="DEV001"
SITE_NAME="Development Site"

# FTP - Sesuaikan dengan setup Anda
ANPR_FTP_HOST="192.168.1.100:21"
ANPR_FTP_USER="ftpuser"
ANPR_FTP_PASS="ftppass"
ANPR_FTP_DIR="/anpr/"
ANPR_FTP_INTERVAL_SEC=5

AXLE_FTP_HOST="192.168.1.100:21"
AXLE_FTP_USER="ftpuser"
AXLE_FTP_PASS="ftppass"
AXLE_FTP_DIR="/axle/"
AXLE_FTP_INTERVAL_SEC=5

# MinIO (jika tidak ada, set kosong untuk disable)
ANPR_MINIO_ENDPOINT="localhost:9000"
ANPR_MINIO_ACCESS_KEY="admin"
ANPR_MINIO_SECRET_KEY="admin12345"
ANPR_MINIO_BUCKET="anpr"
ANPR_MINIO_USE_SSL=false

AXLE_MINIO_ENDPOINT="localhost:9000"
AXLE_MINIO_ACCESS_KEY="admin"
AXLE_MINIO_SECRET_KEY="admin12345"
AXLE_MINIO_BUCKET="axle"
AXLE_MINIO_USE_SSL=false

ATTACHMENT_MINIO_ENDPOINT="localhost:9000"
ATTACHMENT_MINIO_ACCESS_KEY="admin"
ATTACHMENT_MINIO_SECRET_KEY="admin12345"
ATTACHMENT_MINIO_BUCKET="attachment"
ATTACHMENT_MINIO_USE_SSL=false
```

---

## Testing

### 1. Test API - Health Check

```bash
curl http://localhost:4000/health
```

Expected:

```json
{ "status": "ok", "service": "wim-service" }
```

### 2. Test Vehicle Correlation

**Insert test ANPR data:**

```sql
INSERT INTO transact_anpr_capture (id, site_id, plate_no, captured_at)
VALUES (gen_random_uuid(), (SELECT id FROM site WHERE code='DEV001'), 'B 1234 XYZ', NOW());
```

**Wait 2-3 seconds, then insert Axle data:**

```sql
INSERT INTO transact_axle_capture (id, site_id, captured_at, total_axles)
VALUES (gen_random_uuid(), (SELECT id FROM site WHERE code='DEV001'), NOW(), 2);
```

**Check correlation result:**

```sql
SELECT * FROM view_vehicle_complete
WHERE correlation_status = 'MATCHED'
ORDER BY first_detected_at DESC
LIMIT 5;
```

### 3. Check Database

```bash
psql -U wim_user -d wim_db

# Check sites
SELECT * FROM site;

# Check ANPR data
SELECT COUNT(*) FROM transact_anpr_capture;

# Check Axle data
SELECT COUNT(*) FROM transact_axle_capture;

# Check correlation
SELECT correlation_status, COUNT(*)
FROM transact_vehicle
GROUP BY correlation_status;
```

---

## Bagian yang Perlu Diperbaiki

### 🔴 Critical Issues

#### 1. Vehicle Detection - Mock Implementation

**File:** `internal/vision/detector.go`

**Current:**

```go
// TODO: Replace dengan real vehicle detection
// Options: YOLO, OpenCV DNN, TensorFlow, ONNX Runtime
func (d *Detector) Detect(imagePath string) ([]Detection, error) {
    // Mock detection - MUST REPLACE FOR PRODUCTION
    return []Detection{{...}}, nil
}
```

**What to do:**

- Integrate YOLO model untuk vehicle detection
- Atau gunakan OpenCV DNN dengan pre-trained model
- Test dengan real ANPR images
- Update confidence calculation

#### 2. MinIO Error Handling

**File:** `internal/handler/anpr_handler.go`, `internal/handler/axle_handler.go`

**Issue:** Jika MinIO config tidak lengkap atau error, aplikasi tetap jalan tapi tidak upload files.

**What to do:**

- Add proper validation di config loading
- Return error jika MinIO enabled tapi config tidak valid
- Add retry mechanism untuk failed uploads
- Add metrics untuk upload success/failure rate

#### 3. FTP Connection Retry

**File:** `internal/ftpwatcher/watcher.go`

**Issue:** Jika FTP disconnect, tidak ada auto-reconnect mechanism.

**What to do:**

- Implement exponential backoff untuk reconnect
- Add max retry attempts
- Log reconnection attempts
- Alert jika connection gagal berulang kali

#### 4. JWT Token Refresh

**File:** `internal/auth/jwt.go`

**Issue:** Token expired setelah 24 jam, user harus login ulang. Tidak ada refresh token mechanism.

**What to do:**

- Implement refresh token strategy
- Add endpoint `/api/auth/refresh`
- Store refresh token di database
- Auto-refresh before expiry di client side

### 🟡 Medium Priority

#### 5. Vehicle Correlation Configuration

**File:** `migrations/200_vehicle_correlation.sql`

**Issue:** Time window hardcoded 5 seconds di SQL function.

**What to do:**

- Move time window ke environment variable
- Add API endpoint untuk adjust time window per site
- Add monitoring dashboard untuk correlation success rate
- Alert jika match rate < 80%

#### 6. Database Connection Pool

**File:** `cmd/api/main.go`, `cmd/anpr-watcher/main.go`, `cmd/axle-watcher/main.go`

**Issue:** Database connection tidak optimal configured.

**What to do:**

```go
// Add to config
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

#### 7. Rate Limiting

**File:** `internal/api/server.go`

**Issue:** API tidak ada rate limiting, bisa di-abuse.

**What to do:**

- Add rate limiter middleware (contoh: golang.org/x/time/rate)
- Limit per IP atau per user
- Return 429 Too Many Requests
- Whitelist untuk internal services

### 🟢 Low Priority (Nice to Have)

#### 8. Logging Framework

**Current:** `log.Printf` untuk logging

**What to do:**

- Replace dengan structured logging (contoh: zap, logrus)
- Add log levels (DEBUG, INFO, WARN, ERROR)
- Add log rotation
- Send critical logs ke external monitoring (Sentry, etc)

#### 9. Metrics & Monitoring

**What to do:**

- Add Prometheus metrics
- Track: request count, latency, error rate, FTP files processed, correlation rate
- Add Grafana dashboard
- Setup alerts

#### 10. API Documentation

**What to do:**

- Add Swagger/OpenAPI spec
- Generate API docs
- Add example requests/responses
- Host docs di `/api/docs`

#### 11. Unit Tests

**Current:** Tidak ada unit tests

**What to do:**

```bash
# Create test files
internal/auth/jwt_test.go
internal/handler/anpr_handler_test.go
internal/handler/attachment_handler_test.go
internal/ftpwatcher/watcher_test.go

# Run tests
go test ./...
```

#### 12. CI/CD Pipeline

**What to do:**

- Setup GitHub Actions atau GitLab CI
- Auto-build Docker image on push
- Run tests before deploy
- Auto-deploy ke staging/production

#### 13. Configuration Validation

**File:** `internal/config/config.go`

**What to do:**

- Validate semua required env vars di startup
- Return clear error message jika config missing
- Add config validation tests
- Support config file (.yaml/.json) selain env vars

---

## Monitoring

### Match Rate (per site)

```sql
SELECT
    site_code,
    COUNT(*) as total,
    ROUND(100.0 * SUM(CASE WHEN correlation_status = 'MATCHED' THEN 1 ELSE 0 END) / COUNT(*), 2) as match_rate
FROM view_vehicle_complete
GROUP BY site_code;
```

Expected match rate: **85-95%**

### Pending ANPR (waiting for Axle)

```sql
SELECT COUNT(*) FROM view_vehicle_complete
WHERE correlation_status = 'ANPR_ONLY'
AND first_detected_at < NOW() - INTERVAL '1 minute';
```

### 🔍 Monitoring & Logging

Setiap service memiliki logging terpisah. Untuk production, redirect output ke file:

```bash
./bin/wim-api >> logs/api.log 2>&1 &
./bin/wim-anpr-watcher >> logs/anpr.log 2>&1 &
./bin/wim-axle-watcher >> logs/axle.log 2>&1 &
```

---

## Troubleshooting

### 🆘 Service Issues

**Service tidak bisa start:**
```bash
# Check logs
tail -f logs/api.log
tail -f logs/anpr.log
tail -f logs/axle.log

# Check port conflicts
lsof -i :4000  # API port

# Check database connection
psql $DATABASE_URL
```

**Kill all services:**
```bash
pkill -f "wim-api"
pkill -f "wim-anpr-watcher"
pkill -f "wim-axle-watcher"
```

### Low Match Rate (<80%)

**Check:**

1. Time sync (NTP) di sistem ANPR dan Axle
2. Time window setting (default 5s)
3. Both systems capturing data

**Solution:**

```sql
-- Check time distribution
SELECT
    CASE
        WHEN time_diff_seconds < 2 THEN '<2s'
        WHEN time_diff_seconds < 5 THEN '2-5s'
        WHEN time_diff_seconds < 10 THEN '5-10s'
        ELSE '>10s'
    END as time_bucket,
    COUNT(*) as count
FROM view_vehicle_complete
WHERE correlation_status = 'MATCHED'
GROUP BY time_bucket;
```

### FTP Connection Failed

**Solution:**

- Verify credentials di `.env`
- Check network/firewall
- Test manual dengan FileZilla
- Check FTP logs di console output

### MinIO Upload Failed

**Note:** MinIO adalah **optional**. Jika tidak ada MinIO server:

- Set semua `*_MINIO_*` variables ke empty string
- Aplikasi akan tetap jalan, tapi tidak upload file ke storage
- File XML dan metadata tetap tersimpan di database

### Common API Errors

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "Missing or invalid token"
}
```
**Solution:** Pastikan header `Authorization: Bearer <token>` sudah benar.

#### 400 Bad Request (Upload)
```json
{
  "success": false,
  "message": "No image file provided"
}
```
**Solution:**
- Pastikan menggunakan form-data, bukan JSON
- Field name harus `image`
- File adalah gambar yang valid

#### 500 Internal Server Error
**Solution:**
- Check server logs
- Pastikan MinIO accessible
- Pastikan database connection OK
- Pastikan bucket `attachment` sudah dibuat di MinIO

---

## Performance Tips

### 1. Database Indexing

```sql
CREATE INDEX idx_anpr_plate ON transact_anpr_capture(plate_no);
CREATE INDEX idx_anpr_captured ON transact_anpr_capture(captured_at);
CREATE INDEX idx_axle_captured ON transact_axle_capture(captured_at);
CREATE INDEX idx_vehicle_status ON transact_vehicle(correlation_status);
CREATE INDEX idx_vehicle_site ON transact_vehicle(site_id);
```

### 2. FTP Polling Interval

- Adjust `ANPR_FTP_INTERVAL_SEC` sesuai traffic
- Default 5 detik recommended
- Jangan terlalu cepat (<2 detik) karena bisa overload

### 3. Database Connection Pool

```sql
-- Check active connections
SELECT count(*) FROM pg_stat_activity;

-- Recommended settings di PostgreSQL
max_connections = 100
shared_buffers = 256MB
```

---

## Project Structure

```
wim-service/
├── cmd/
│   ├── api/                   # API Server
│   ├── anpr-watcher/          # ANPR FTP Watcher
│   └── axle-watcher/          # AXLE FTP Watcher
├── internal/
│   ├── api/                   # REST API handlers
│   ├── auth/                  # JWT Authentication
│   ├── config/                # Configuration loader
│   ├── ftpwatcher/            # FTP monitoring
│   └── handler/               # Business logic (ANPR, Axle, Attachment)
├── migrations/
│   └── 200_vehicle_correlation.sql
├── .env.example               # Environment template
├── portainer-stack.yml        # Portainer deployment
├── docker-compose.yml         # Docker Compose setup
├── start-all.sh               # Start all services script
└── README.md                  # This file
```

---

## Production Deployment Checklist

- [ ] Change `JWT_SECRET` ke strong random string (min 32 chars)
- [ ] Set proper database credentials
- [ ] Enable SSL untuk database connection (`sslmode=require`)
- [ ] Configure MinIO dengan production credentials
- [ ] Setup backup untuk PostgreSQL (daily)
- [ ] Enable HTTPS untuk API server
- [ ] Setup reverse proxy (Nginx/Traefik)
- [ ] Configure firewall rules
- [ ] Setup monitoring (Prometheus + Grafana)
- [ ] Configure log rotation
- [ ] Test failover scenarios
- [ ] Document deployment procedures
- [ ] Setup CI/CD pipeline

---

## 🔑 Default Credentials

**Admin User:**
- Username: `admin`
- Email: `admin@wim.local`
- Password: `admin123`

**Note:** User ini harus sudah dibuat di database melalui migration atau seeding.

---

## License

Proprietary software.

## Author

**Taufan Senagumelar**

- GitHub: [@tsenagumelar](https://github.com/tsenagumelar)
- Repository: [wim-service](https://github.com/tsenagumelar/wim-service)

---

**Made with ❤️ for WIM Service**
