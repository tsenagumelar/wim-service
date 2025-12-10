# WIM Service

**Weigh In Motion Service** - Service untuk monitoring kendaraan melalui ANPR/CCTV Vidar dengan fitur:

- 🚗 ANPR (Automatic Number Plate Recognition) Processing
- ⚖️ AXLE Weight Processing
- 📏 Vehicle Dimension Detection
- 🔐 REST API dengan JWT Authentication
- ☁️ MinIO Storage Integration

---

## 📋 Features

### 1. **ANPR Processing**

- Monitoring FTP untuk file XML dan gambar ANPR
- Upload otomatis ke MinIO storage
- Simpan metadata ke database PostgreSQL
- Support multiple camera locations

### 2. **AXLE Weight Processing**

- Monitoring FTP untuk data axle weight
- Process XML data dari sensor WIM
- Integration dengan MinIO storage

### 3. **Vehicle Dimension Detection**

- Deteksi dimensi kendaraan (panjang, lebar, tinggi) dari gambar ANPR
- Camera calibration untuk akurasi perhitungan
- Vehicle classification berdasarkan dimensi
- Automatic integration dengan ANPR processing

### 4. **REST API with Authentication**

- JWT token-based authentication (expired 3x24 jam)
- User management
- Protected endpoints
- CORS enabled

---

## 🚀 Quick Start

### Prerequisites

- Go 1.24+ (for local development)
- Docker & Docker Compose (for containerized deployment)
- PostgreSQL database
- MinIO server (optional)
- FTP server dengan data ANPR/AXLE

### Option 1: Portainer Stack Deployment (Recommended)

**Prerequisites:**

- Portainer sudah terinstall
- PostgreSQL database sudah tersedia
- MinIO storage sudah tersedia (optional)
- FTP server sudah configured

**Steps:**

1. **Build dan Push Docker Image**

```bash
# Clone repository
git clone https://github.com/tsenagumelar/wim-service.git
cd wim-service

# Build image dengan tag
docker build -t tsenagumelar/wim-service:latest .

# Push ke Docker Hub (atau private registry)
docker push tsenagumelar/wim-service:latest

# Atau dengan versioning
docker build -t tsenagumelar/wim-service:v1.0.0 .
docker push tsenagumelar/wim-service:v1.0.0
```

2. **Deploy di Portainer**

- Login ke Portainer UI
- Pilih **Stacks** → **Add Stack**
- Nama stack: `wim-service`
- Build method: **Web editor**
- Copy isi file `portainer-stack.yml` ke editor
- **PENTING:** Edit environment variables sesuai setup Anda:
  - `DATABASE_URL` - connection string PostgreSQL Anda
  - `JWT_SECRET` - secret key untuk JWT (min 32 karakter)
  - `ANPR_FTP_*` - konfigurasi FTP server ANPR
  - `AXLE_FTP_*` - konfigurasi FTP server AXLE
  - `*_MINIO_*` - konfigurasi MinIO storage
  - `CAMERA_*` - parameter kalibrasi camera
- Klik **Deploy the stack**

3. **Verifikasi Deployment**

```bash
# Check container logs di Portainer UI atau CLI
docker logs wim-service

# Test API endpoint
curl http://localhost:3000/health

# Response:
# {"status":"ok","service":"wim-service"}
```

**Default credentials:**

- Username: `admin`
- Password: `admin123`

**File yang dibutuhkan:**

- `portainer-stack.yml` - Stack definition untuk Portainer
- Docker image: `tsenagumelar/wim-service:latest`

### Option 2: Local Development

1. **Clone repository**

```bash
git clone https://github.com/tsenagumelar/wim-service.git
cd wim-service
```

2. **Install dependencies**

```bash
go mod download
```

3. **Setup environment**

```bash
cp .env.example .env
# Edit .env dengan konfigurasi Anda
```

4. **Run service**

```bash
go run main.go
```

Output:

```
[MAIN] WIM Service starting...
[MAIN] Starting API Server...
[MAIN] API Server: http://localhost:3000
[MAIN] Login: POST http://localhost:3000/api/auth/login
[MAIN] Default credentials - username: admin, password: admin123
[MAIN] Starting ANPR watcher...
[MAIN] Starting AXLE watcher...
[MAIN] All services started successfully!
```

### Option 3: Docker Compose (Development)

Untuk development dengan PostgreSQL dan MinIO lokal:

```bash
# Clone repository
git clone https://github.com/tsenagumelar/wim-service.git
cd wim-service

# Edit docker-compose.yml sesuai kebutuhan
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f wim-service

# Stop services
docker-compose down
```

**Services yang berjalan:**

- PostgreSQL: `localhost:5432`
- MinIO API: `localhost:9000`
- MinIO Console: `localhost:9001`
- WIM Service API: `localhost:3000`

---

## ⚙️ Configuration

### Environment Variables

Create `.env` file:

```bash
# Database
DATABASE_URL="postgres://user:password@host:5432/dbname?sslmode=disable"

# API Server
API_PORT=3000
JWT_SECRET="your-secret-key-change-in-production"

# ANPR FTP Configuration
ANPR_FTP_HOST="192.168.1.100:21"
ANPR_FTP_USER="ftpuser"
ANPR_FTP_PASS="ftppass"
ANPR_FTP_DIR="/anpr/"
ANPR_FTP_INTERVAL_SEC=5

# AXLE FTP Configuration
AXLE_FTP_HOST="192.168.1.100:21"
AXLE_FTP_USER="ftpuser"
AXLE_FTP_PASS="ftppass"
AXLE_FTP_DIR="/axle/"
AXLE_FTP_INTERVAL_SEC=5

# ANPR MinIO Storage
ANPR_MINIO_ENDPOINT="s3.example.com"
ANPR_MINIO_ACCESS_KEY="admin"
ANPR_MINIO_SECRET_KEY="admin12345"
ANPR_MINIO_BUCKET="anpr"
ANPR_MINIO_USE_SSL=true

# AXLE MinIO Storage
AXLE_MINIO_ENDPOINT="s3.example.com"
AXLE_MINIO_ACCESS_KEY="admin"
AXLE_MINIO_SECRET_KEY="admin12345"
AXLE_MINIO_BUCKET="axle"
AXLE_MINIO_USE_SSL=true

# Vehicle Dimension Detection
DIMENSION_ENABLED=true
DIMENSION_THRESHOLD=0.5
DIMENSION_MODEL_PATH=

# Camera Calibration (PENTING untuk akurasi!)
CAMERA_IMAGE_WIDTH=1920
CAMERA_IMAGE_HEIGHT=1080
CAMERA_FOCAL_LENGTH=1000.0
CAMERA_HEIGHT_METERS=6.0
CAMERA_TILT_ANGLE=30.0
CAMERA_REF_PIXEL_LENGTH=1500
CAMERA_REF_REAL_LENGTH=5.0
CAMERA_REF_DISTANCE=15.0
```

---

## 📡 REST API Documentation

### Authentication

#### 1. Login

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
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-12-13T14:30:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@wim.local",
      "role": "admin"
    }
  }
}
```

**Token Expiration:** 3x24 jam (72 hours)

#### 2. Get Profile (Protected)

```bash
GET /api/auth/profile
Authorization: Bearer <your-jwt-token>
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "admin",
    "email": "admin@wim.local",
    "role": "admin"
  }
}
```

#### 3. Health Check

```bash
GET /health
```

**Response:**

```json
{
  "status": "ok",
  "service": "wim-service"
}
```

---

## 📏 Vehicle Dimension Detection

### Cara Kerja

1. **Vehicle Detection**: Detect kendaraan dalam gambar (bounding box)
2. **Camera Calibration**: Konversi pixel ke meter
3. **Dimension Calculation**: Hitung dimensi (panjang, lebar, tinggi)
4. **Vehicle Classification**: Klasifikasi jenis kendaraan

### Kalibrasi Camera (PENTING!)

Untuk hasil akurat, lakukan kalibrasi:

**Step 1: Ukur Parameter Camera**

- Tinggi camera dari jalan (meter)
- Sudut tilt camera (derajat)
- Focal length (dari spesifikasi camera)

**Step 2: Ukur Reference Object**

- Pilih objek referensi (contoh: garis jalan 5 meter)
- Buka gambar ANPR di image editor
- Ukur berapa pixel panjang objek tersebut
- Ukur jarak camera ke objek

**Step 3: Update .env**

```bash
CAMERA_REF_PIXEL_LENGTH=1500  # Hasil ukuran dalam pixel
CAMERA_REF_REAL_LENGTH=5.0     # Ukuran sebenarnya (meter)
CAMERA_REF_DISTANCE=15.0       # Jarak camera ke objek
```

### Vehicle Classification

| Class      | Length   | Width    | Description           |
| ---------- | -------- | -------- | --------------------- |
| motorcycle | < 2.5m   | < 1.5m   | Sepeda Motor          |
| sedan      | 2.5-5.5m | < 2.0m   | Mobil Penumpang       |
| suv        | 4.0-6.0m | 1.8-2.2m | SUV/Minivan           |
| truck      | 5.5-12m  | -        | Truk/Kendaraan Barang |
| bus        | > 7.0m   | > 2.0m   | Bus/Kendaraan Besar   |

### Testing Dimension Detection

```bash
# Copy gambar test ke folder root
cp /path/to/anpr/image.jpg ./test_image.jpg

# Run test
go run test_dimension.go
```

**Output:**

```
=== Processing Image: test_image.jpg ===
[DIMENSION] Detected 1 vehicle(s)

Vehicle 1:
  Length: 4.52 meters
  Width: 1.85 meters
  Height: 1.11 meters (estimated)
  Distance from camera: 12.30 meters
  Confidence: 67%
  Classification: Mobil Penumpang / Sedan
```

---

## 🗄️ Database Schema

### Table: users

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Table: transact_anpr_capture

```sql
-- Existing ANPR table with additional dimension columns
ALTER TABLE transact_anpr_capture
ADD COLUMN vehicle_length DECIMAL(10, 3),
ADD COLUMN vehicle_width DECIMAL(10, 3),
ADD COLUMN vehicle_height DECIMAL(10, 3),
ADD COLUMN vehicle_class VARCHAR(50),
ADD COLUMN dimension_confidence DECIMAL(5, 4);
```

### Table: vehicle_dimensions

```sql
CREATE TABLE vehicle_dimensions (
    id SERIAL PRIMARY KEY,
    image_path VARCHAR(500),
    length_meters DECIMAL(10, 3),
    width_meters DECIMAL(10, 3),
    height_meters DECIMAL(10, 3),
    distance_meters DECIMAL(10, 3),
    confidence DECIMAL(5, 4),
    vehicle_class VARCHAR(50),
    class_description VARCHAR(200),
    center_x INT,
    center_y INT,
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 📂 Project Structure

```
wim-service/
├── main.go                          # Entry point - All services
├── test_dimension.go                # Test dimension detection
├── .env                             # Environment configuration
├── go.mod                           # Go dependencies
├── internal/
│   ├── api/                         # REST API
│   │   ├── server.go                # API server setup
│   │   └── auth_handler.go          # Auth endpoints
│   ├── auth/                        # Authentication
│   │   ├── types.go                 # Auth types
│   │   ├── jwt.go                   # JWT helpers
│   │   └── service.go               # Auth service
│   ├── config/                      # Configuration
│   │   ├── config.go                # Config loader
│   │   └── calibration.go           # Camera calibration
│   ├── ftpwatcher/                  # FTP monitoring
│   │   └── watcher.go               # FTP watcher
│   ├── handler/                     # Business logic
│   │   ├── anpr_handler.go          # ANPR processing
│   │   ├── axle_handler.go          # AXLE processing
│   │   └── dimension_handler.go     # Dimension processing
│   └── vision/                      # Computer vision
│       ├── types.go                 # Vision types
│       ├── detector.go              # Vehicle detection
│       ├── calibration.go           # Camera calibration
│       └── dimension.go             # Dimension calculation
└── README.md                        # This file
```

---

## 🔧 Advanced Usage

### Custom Vehicle Detection

Saat ini menggunakan mock detection. Untuk production, integrate dengan:

- **YOLO** (You Only Look Once)
- **OpenCV DNN**
- **TensorFlow**
- **ONNX Runtime**

### Multiple Camera Support

Service sudah support multiple camera:

- Setiap camera punya konfigurasi FTP sendiri
- Kalibrasi per camera location
- Automatic classification per lokasi

### Monitoring & Logging

Semua activity di-log dengan format:

```
[COMPONENT] Message
```

Components:

- `[MAIN]` - Main application
- `[API]` - API server
- `[AUTH]` - Authentication
- `[ANPR]` - ANPR processing
- `[AXLE]` - AXLE processing
- `[DIMENSION]` - Dimension detection
- `[CONFIG]` - Configuration

---

## 🐛 Troubleshooting

### 1. Dimensi Tidak Akurat

**Solusi:**

- Check parameter kalibrasi di `.env`
- Ukur ulang reference object dengan teliti
- Pastikan `CAMERA_REF_PIXEL_LENGTH` sesuai dengan gambar
- Test dengan kendaraan yang dimensinya sudah diketahui

### 2. API Server Error

**Solusi:**

- Check `API_PORT` tidak digunakan aplikasi lain
- Verifikasi database connection string
- Check JWT_SECRET sudah di-set

### 3. FTP Connection Failed

**Solusi:**

- Verifikasi FTP credentials
- Check firewall/network connectivity
- Test FTP connection manual dengan FileZilla

### 4. Token Expired

**Solusi:**

- Login ulang untuk mendapatkan token baru
- Token valid 72 jam (3x24 jam)
- Check system time/timezone

---

## 📈 Performance Tips

1. **Database Indexing**

```sql
CREATE INDEX idx_anpr_plate ON transact_anpr_capture(plate_no);
CREATE INDEX idx_anpr_captured ON transact_anpr_capture(captured_at);
CREATE INDEX idx_dimensions_processed ON vehicle_dimensions(processed_at);
```

2. **FTP Polling Interval**

- Adjust `ANPR_FTP_INTERVAL_SEC` sesuai traffic
- Default 5 detik cukup untuk most cases

3. **MinIO Connection Pool**

- Service sudah optimize connection reuse
- Monitor MinIO performance dashboard

---

## 🤝 Contributing

1. Fork repository
2. Create feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit changes (`git commit -m 'Add AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Open Pull Request

---

## 📝 License

This project is proprietary software.

---

## 👤 Author

**Taufan Senagumelar**

- GitHub: [@tsenagumelar](https://github.com/tsenagumelar)
- Repository: [wim-service](https://github.com/tsenagumelar/wim-service)

---

## 📞 Support

Untuk pertanyaan dan support, silakan buat issue di GitHub repository.

---

**Made with ❤️ for WIM Service**
