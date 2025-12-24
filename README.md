# WIM Service

**Weigh In Motion Service** - Service monitoring kendaraan dengan ANPR, AXLE Weight Sensor, dan RTSP Camera Streaming.

## Tentang Aplikasi

WIM Service adalah sistem monitoring kendaraan yang menggabungkan data dari berbagai sensor:

- **ANPR (Automatic Number Plate Recognition)** - Mendeteksi plat nomor kendaraan
- **AXLE Sensor** - Mengukur jumlah axle dan dimensi kendaraan
- **RTSP Streaming** - Live streaming dari IP camera
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

### 4. RTSP Camera Streaming

- Stream dari IP camera via WebSocket
- Support multiple camera feeds
- H.264 encoding
- Ready untuk Next.js integration

### 5. REST API + JWT Authentication

- Token-based auth (expired 72 jam)
- Protected endpoints
- CORS enabled

### 6. Multi-Site Architecture

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

3. **Run**

```bash
go run main.go
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

# RTSP Streaming (optional)
RTSP_ENABLED=true
RTSP_URL="rtsp://admin:admin@192.168.1.18:554/stream1"
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

## RTSP Streaming

### Cara Kerja

Server Go convert **RTSP → H.264 → WebSocket** untuk browser.

### WebSocket Endpoint

```
ws://localhost:4000/api/rtsp/stream/{stream_id}/ws
```

Default stream ID: `camera1` (dari env `RTSP_URL`)

### API Endpoints

| Method | Endpoint                      | Description         |
| ------ | ----------------------------- | ------------------- |
| GET    | `/api/rtsp/streams`           | List all streams    |
| GET    | `/api/rtsp/streams/:id/info`  | Stream info         |
| POST   | `/api/rtsp/streams`           | Add new stream      |
| POST   | `/api/rtsp/streams/:id/start` | Start stream        |
| POST   | `/api/rtsp/streams/:id/stop`  | Stop stream         |
| WS     | `/api/rtsp/stream/:id/ws`     | WebSocket streaming |

### Test dari Browser Console

```javascript
const ws = new WebSocket("ws://localhost:4000/api/rtsp/stream/camera1/ws");
ws.onopen = () => console.log("✅ Connected");
ws.onmessage = (e) => {
  const data = JSON.parse(e.data);
  console.log(data.type); // 'stream_started' or 'frame'
};
```

---

## REST API

### Authentication

**Login:**

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
    "expires_at": "2025-12-21T10:00:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

**Get Profile (Protected):**

```bash
GET /api/auth/profile
Authorization: Bearer <token>
```

**Health Check:**

```bash
GET /health
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

## Project Structure

```
wim-service/
├── main.go                    # Entry point
├── .env.example               # Environment template
├── portainer-stack.yml        # Portainer deployment
├── migrations/
│   └── 200_vehicle_correlation.sql
├── internal/
│   ├── api/                   # REST API
│   ├── auth/                  # JWT Authentication
│   ├── config/                # Configuration loader
│   ├── ftpwatcher/            # FTP monitoring
│   ├── handler/               # Business logic (ANPR, Axle)
│   └── rtspstream/            # RTSP streaming
├── README.md                  # This file
└── DEVELOPMENT_GUIDE.md       # Development guide
```

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

---

## Troubleshooting

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

### RTSP Stream Timeout

**Solution:**

- Test RTSP URL dengan VLC: `vlc rtsp://...`
- Verify camera IP reachable: `ping 192.168.1.18`
- Check credentials dan port
- Verify `RTSP_ENABLED=true` di `.env`

### MinIO Upload Failed

**Note:** MinIO adalah **optional**. Jika tidak ada MinIO server:

- Set semua `*_MINIO_*` variables ke empty string
- Aplikasi akan tetap jalan, tapi tidak upload file ke storage
- File XML dan metadata tetap tersimpan di database

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

## Development

Untuk development guide lengkap, lihat [DEVELOPMENT_GUIDE.md](DEVELOPMENT_GUIDE.md)

Includes:

- Setup development environment
- Cara menjalankan aplikasi (4 options)
- Testing (API, Correlation, RTSP, Database)
- **Bagian yang perlu diperbaiki** (Critical/Medium/Low priority)
- Next.js frontend integration dengan code examples

---

## License

Proprietary software.

## Author

**Taufan Senagumelar**

- GitHub: [@tsenagumelar](https://github.com/tsenagumelar)
- Repository: [wim-service](https://github.com/tsenagumelar/wim-service)

---

**Made with ❤️ for WIM Service**
