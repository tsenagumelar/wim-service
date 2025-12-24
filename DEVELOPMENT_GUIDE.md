# Development Guide - WIM Service

Panduan lengkap untuk setup development, menjalankan aplikasi, dan improvement yang perlu dilakukan.

---

## Table of Contents

- [Setup Development Environment](#setup-development-environment)
- [Cara Menjalankan Aplikasi](#cara-menjalankan-aplikasi)
- [Testing](#testing)
- [Bagian yang Perlu Diperbaiki](#bagian-yang-perlu-diperbaiki)
- [Next.js Frontend Integration](#nextjs-frontend-integration)

---

## Setup Development Environment

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
# Create buckets: anpr, axle
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

# RTSP Streaming (optional)
RTSP_ENABLED=true
RTSP_URL="rtsp://admin:admin@192.168.1.18:554/stream1"
```

---

## Cara Menjalankan Aplikasi

### Option 1: Run Langsung (Development)

```bash
go run main.go
```

**Output:**

```
[MAIN] WIM Service starting...
[MAIN] Site: DEV001 - Development Site
[CONFIG] Loaded configuration
[MAIN] Starting API Server...
[MAIN] API Server: http://localhost:4000
[MAIN] Login: POST http://localhost:4000/api/auth/login
[MAIN] Default credentials - username: admin, password: admin123
[MAIN] Starting ANPR watcher...
[MAIN] Starting AXLE watcher...
[MAIN] ✅ RTSP stream registered - Stream ID: camera1
[MAIN] All services started successfully!
```

### Option 2: Build & Run

```bash
# Build
go build -o wim-service main.go

# Run
./wim-service
```

### Option 3: Docker Compose

```bash
# Start all services (PostgreSQL + MinIO + WIM Service)
docker-compose up -d

# View logs
docker-compose logs -f wim-service

# Stop
docker-compose down
```

### Option 4: Portainer Stack

1. Build image: `docker build -t tsenagumelar/wim-service:dev .`
2. Push ke registry atau load ke Portainer
3. Deploy menggunakan `portainer-stack.yml`

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

### 2. Test API - Login

```bash
curl -X POST http://localhost:4000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

Expected:

```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2025-12-21T10:00:00Z",
    "user": { "id": 1, "username": "admin", "role": "admin" }
  }
}
```

### 3. Test API - Get Profile

```bash
# Ganti <TOKEN> dengan token dari login
curl http://localhost:4000/api/auth/profile \
  -H "Authorization: Bearer <TOKEN>"
```

### 4. Test Vehicle Correlation

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

### 5. Test RTSP Streaming

**List streams:**

```bash
curl http://localhost:4000/api/rtsp/streams
```

**Get stream info:**

```bash
curl http://localhost:4000/api/rtsp/streams/camera1/info
```

**Test WebSocket (browser console):**

```javascript
const ws = new WebSocket("ws://localhost:4000/api/rtsp/stream/camera1/ws");
ws.onopen = () => console.log("✅ Connected");
ws.onmessage = (e) => {
  const data = JSON.parse(e.data);
  console.log(data.type, data.type === "frame" ? "Frame received" : data);
};
ws.onerror = (e) => console.error("❌ Error", e);
ws.onclose = () => console.log("🔌 Disconnected");
```

### 6. Check Database

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

**Issue:** Token expired setelah 72 jam, user harus login ulang. Tidak ada refresh token mechanism.

**What to do:**

- Implement refresh token strategy
- Add endpoint `/api/auth/refresh`
- Store refresh token di database
- Auto-refresh before expiry di client side

### 🟡 Medium Priority

#### 5. RTSP H.264 Decoder

**File:** Frontend integration

**Issue:** Backend kirim raw H.264 frames, frontend belum ada decoder.

**What to do:**

- Integrate broadway.js atau jsmpeg di Next.js
- Add canvas rendering untuk video display
- Test dengan real camera stream
- Add video quality controls (low/medium/high)

#### 6. Vehicle Correlation Configuration

**File:** `migrations/200_vehicle_correlation.sql`

**Issue:** Time window hardcoded 5 seconds di SQL function.

**What to do:**

- Move time window ke environment variable
- Add API endpoint untuk adjust time window per site
- Add monitoring dashboard untuk correlation success rate
- Alert jika match rate < 80%

#### 7. Database Connection Pool

**File:** `main.go`

**Issue:** Database connection tidak optimal configured.

**What to do:**

```go
// Add to config
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

#### 8. Rate Limiting

**File:** `internal/api/server.go`

**Issue:** API tidak ada rate limiting, bisa di-abuse.

**What to do:**

- Add rate limiter middleware (contoh: golang.org/x/time/rate)
- Limit per IP atau per user
- Return 429 Too Many Requests
- Whitelist untuk internal services

### 🟢 Low Priority (Nice to Have)

#### 9. Logging Framework

**Current:** `fmt.Printf` untuk logging

**What to do:**

- Replace dengan structured logging (contoh: zap, logrus)
- Add log levels (DEBUG, INFO, WARN, ERROR)
- Add log rotation
- Send critical logs ke external monitoring (Sentry, etc)

#### 10. Metrics & Monitoring

**What to do:**

- Add Prometheus metrics
- Track: request count, latency, error rate, FTP files processed, correlation rate
- Add Grafana dashboard
- Setup alerts

#### 11. API Documentation

**What to do:**

- Add Swagger/OpenAPI spec
- Generate API docs
- Add example requests/responses
- Host docs di `/api/docs`

#### 12. Unit Tests

**Current:** Tidak ada unit tests

**What to do:**

```bash
# Create test files
internal/auth/jwt_test.go
internal/handler/anpr_handler_test.go
internal/ftpwatcher/watcher_test.go

# Run tests
go test ./...
```

#### 13. CI/CD Pipeline

**What to do:**

- Setup GitHub Actions atau GitLab CI
- Auto-build Docker image on push
- Run tests before deploy
- Auto-deploy ke staging/production

#### 14. Configuration Validation

**File:** `internal/config/config.go`

**What to do:**

- Validate semua required env vars di startup
- Return clear error message jika config missing
- Add config validation tests
- Support config file (.yaml/.json) selain env vars

---

## Next.js Frontend Integration

### Setup Next.js Project

```bash
# Create Next.js app
npx create-next-app@latest wim-frontend
cd wim-frontend

# Pilih:
# ✅ TypeScript
# ✅ Tailwind CSS
# ✅ App Router
```

### Project Structure

```
wim-frontend/
├── app/
│   ├── camera/
│   │   └── page.tsx          # Camera monitoring page
│   ├── dashboard/
│   │   └── page.tsx          # Vehicle data dashboard
│   └── layout.tsx            # Root layout
├── components/
│   ├── RTSPPlayer.tsx        # Single camera player
│   ├── DualCamera.tsx        # Dual camera view
│   └── VehicleTable.tsx      # Vehicle data table
├── hooks/
│   ├── useRTSPStream.tsx     # WebSocket RTSP hook
│   └── useAuth.tsx           # JWT auth hook
└── lib/
    └── api.ts                # API client
```

### 1. API Client Setup

**File:** `lib/api.ts`

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:4000";

export async function login(username: string, password: string) {
  const res = await fetch(`${API_BASE}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  return res.json();
}

export async function getProfile(token: string) {
  const res = await fetch(`${API_BASE}/api/auth/profile`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  return res.json();
}

export async function getVehicles(token: string, limit = 100) {
  // TODO: Add API endpoint di Go server
  const res = await fetch(`${API_BASE}/api/vehicles?limit=${limit}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  return res.json();
}
```

### 2. RTSP Player Hook

**File:** `hooks/useRTSPStream.tsx`

```typescript
"use client";

import { useEffect, useRef, useState } from "react";

interface UseRTSPStreamReturn {
  connected: boolean;
  frameCount: number;
  fps: number;
  error: string | null;
}

export function useRTSPStream(
  streamId: string,
  serverUrl: string = "ws://localhost:4000"
): UseRTSPStreamReturn {
  const [connected, setConnected] = useState(false);
  const [frameCount, setFrameCount] = useState(0);
  const [fps, setFps] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const startTimeRef = useRef<number>(0);

  useEffect(() => {
    const wsUrl = `${serverUrl}/api/rtsp/stream/${streamId}/ws`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;
    startTimeRef.current = Date.now();

    ws.onopen = () => {
      setConnected(true);
      setError(null);
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === "frame") {
          setFrameCount((prev) => {
            const newCount = prev + 1;
            const elapsed = (Date.now() - startTimeRef.current) / 1000;
            setFps(Number((newCount / elapsed).toFixed(1)));
            return newCount;
          });
        }
      } catch (err) {
        console.error("Parse error:", err);
      }
    };

    ws.onerror = () => {
      setError("Connection error");
      setConnected(false);
    };

    ws.onclose = () => {
      setConnected(false);
    };

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.close();
      }
    };
  }, [streamId, serverUrl]);

  return { connected, frameCount, fps, error };
}
```

### 3. RTSP Player Component

**File:** `components/RTSPPlayer.tsx`

```typescript
"use client";

import { useRTSPStream } from "@/hooks/useRTSPStream";

interface RTSPPlayerProps {
  streamId: string;
  title?: string;
  serverUrl?: string;
}

export default function RTSPPlayer({
  streamId,
  title,
  serverUrl = "ws://localhost:4000",
}: RTSPPlayerProps) {
  const { connected, frameCount, fps, error } = useRTSPStream(
    streamId,
    serverUrl
  );

  return (
    <div className="border rounded-lg overflow-hidden shadow-lg">
      {/* Header */}
      <div className="bg-gray-800 text-white p-4 flex items-center justify-between">
        <div>
          <h3 className="font-bold text-lg">{title || streamId}</h3>
          <p className="text-sm text-gray-400">Stream ID: {streamId}</p>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right">
            <p className="text-sm text-gray-400">Frames</p>
            <p className="font-mono text-xl">{frameCount.toLocaleString()}</p>
          </div>
          <div className="text-right">
            <p className="text-sm text-gray-400">FPS</p>
            <p className="font-mono text-xl">{fps}</p>
          </div>
          <div
            className={`px-3 py-1 rounded text-sm font-medium ${
              connected ? "bg-green-500" : "bg-red-500"
            }`}
          >
            {connected ? "🟢 Live" : "🔴 Offline"}
          </div>
        </div>
      </div>

      {/* Video Area */}
      <div className="bg-black aspect-video flex items-center justify-center">
        {error ? (
          <div className="text-center text-red-400">
            <p className="text-xl mb-2">⚠️</p>
            <p>{error}</p>
          </div>
        ) : connected ? (
          <div className="text-center text-white">
            <p className="text-6xl mb-4">📹</p>
            <p className="text-xl">Receiving H.264 Stream</p>
            <p className="text-sm text-gray-400 mt-2">Frame #{frameCount}</p>
            <p className="text-xs text-gray-500 mt-4">
              Note: Install broadway.js or jsmpeg for video decode
            </p>
          </div>
        ) : (
          <div className="text-center text-gray-400">
            <p className="text-xl mb-2">🔄</p>
            <p>Connecting...</p>
          </div>
        )}
      </div>
    </div>
  );
}
```

### 4. Dual Camera Page

**File:** `app/camera/page.tsx`

```typescript
"use client";

import RTSPPlayer from "@/components/RTSPPlayer";

export default function CameraPage() {
  return (
    <div className="min-h-screen bg-gray-100 p-8">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-4xl font-bold text-gray-900 mb-8">
          📹 RTSP Camera Monitoring
        </h1>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <RTSPPlayer
            streamId="camera1"
            title="Camera 1 - Entry"
            serverUrl="ws://localhost:4000"
          />
          <RTSPPlayer
            streamId="camera2"
            title="Camera 2 - Exit"
            serverUrl="ws://localhost:4000"
          />
        </div>
      </div>
    </div>
  );
}
```

### 5. Environment Variables

**File:** `.env.local`

```bash
NEXT_PUBLIC_API_URL=http://localhost:4000
NEXT_PUBLIC_WS_URL=ws://localhost:4000
```

### 6. Run Next.js

```bash
npm run dev
# Open: http://localhost:3001/camera
```

---

## Development Workflow

### 1. Feature Development

```bash
# Create feature branch
git checkout -b feature/add-api-endpoint

# Make changes
# ...

# Test locally
go run main.go

# Commit
git add .
git commit -m "Add new API endpoint"

# Push
git push origin feature/add-api-endpoint

# Create PR
```

### 2. Database Changes

```bash
# Create new migration file
touch migrations/210_new_feature.sql

# Write SQL
# ...

# Apply migration
psql -U wim_user -d wim_db -f migrations/210_new_feature.sql

# Test
# ...

# Commit
git add migrations/210_new_feature.sql
git commit -m "Add new feature migration"
```

### 3. Hot Reload (Development)

Install air untuk hot reload:

```bash
go install github.com/cosmtrek/air@latest

# Create .air.toml
# ...

# Run dengan hot reload
air
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

## Support & Contribution

**Issues:** [GitHub Issues](https://github.com/tsenagumelar/wim-service/issues)

**Pull Requests:** Welcome!

**Contact:** [@tsenagumelar](https://github.com/tsenagumelar)

---

**Happy Coding! 🚀**
