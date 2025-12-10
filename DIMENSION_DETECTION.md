# Vehicle Dimension Detection - WIM Service

## Overview

Service ini sudah dilengkapi dengan fitur **Vehicle Dimension Detection** yang dapat mengidentifikasi dimensi kendaraan (panjang, lebar, tinggi) dari gambar ANPR/CCTV.

## Cara Kerja

1. **Vehicle Detection**: Mendeteksi kendaraan dalam gambar dan mendapatkan bounding box
2. **Camera Calibration**: Mengkonversi ukuran pixel ke ukuran fisik (meter) menggunakan parameter kalibrasi
3. **Dimension Calculation**: Menghitung dimensi kendaraan berdasarkan posisi dan ukuran dalam gambar
4. **Vehicle Classification**: Mengklasifikasikan jenis kendaraan berdasarkan dimensi

## Struktur Package

```
internal/
├── vision/
│   ├── detector.go       # Vehicle detection dan bounding box
│   ├── calibration.go    # Camera calibration dan konversi pixel-to-meter
│   └── dimension.go      # Dimension calculation dan classification
├── handler/
│   └── dimension_handler.go  # Integration dengan ANPR processor
└── config/
    ├── config.go         # Configuration loading
    └── calibration.go    # Calibration config helper
```

## Konfigurasi

### 1. Enable Dimension Detection

Edit `.env` file:

```bash
DIMENSION_ENABLED=true
DIMENSION_THRESHOLD=0.5
```

### 2. Camera Calibration (PENTING!)

Untuk hasil yang akurat, Anda HARUS melakukan kalibrasi camera:

```bash
# Resolusi gambar dari camera
CAMERA_IMAGE_WIDTH=1920
CAMERA_IMAGE_HEIGHT=1080

# Focal length (adjust sesuai camera)
CAMERA_FOCAL_LENGTH=1000.0

# Tinggi camera dari tanah (dalam meter)
CAMERA_HEIGHT_METERS=6.0

# Sudut kemiringan camera (0° = horizontal, 90° = tegak lurus ke bawah)
CAMERA_TILT_ANGLE=30.0
```

### 3. Reference Object Calibration

Gunakan objek referensi di lapangan untuk kalibrasi:

```bash
# Contoh: Garis marka jalan sepanjang 5 meter
# Ukur berapa pixel yang diisi garis tersebut dalam gambar
CAMERA_REF_PIXEL_LENGTH=200

# Panjang sebenarnya dalam meter
CAMERA_REF_REAL_LENGTH=5.0

# Jarak dari camera ke objek referensi
CAMERA_REF_DISTANCE=10.0
```

## Cara Melakukan Kalibrasi

### Step 1: Ukur Parameter Camera

1. **Camera Height**: Ukur tinggi camera dari permukaan jalan
2. **Tilt Angle**: Ukur sudut kemiringan camera (gunakan inclinometer atau estimate)
3. **Focal Length**: Check spesifikasi camera atau gunakan nilai default (1000 px)

### Step 2: Ukur Reference Object

1. Pilih objek yang terlihat jelas di gambar (garis jalan, tanda parkir, dll)
2. Ukur panjang sebenarnya objek tersebut (misal: 5 meter)
3. Buka gambar dari camera di image editor
4. Hitung berapa pixel yang diisi objek tersebut
5. Ukur jarak dari camera ke objek

### Step 3: Test Calibration

```bash
# Set DIMENSION_ENABLED=true di .env
# Jalankan service
go run main.go

# Upload test image dan check log untuk hasil dimensi
```

## Penggunaan

### Otomatis dengan ANPR Processor

Jika `DIMENSION_ENABLED=true`, setiap gambar ANPR yang diproses akan otomatis:

1. Deteksi kendaraan
2. Hitung dimensi
3. Simpan ke database (`vehicle_dimensions` table)
4. Update `transact_anpr_capture` dengan kolom dimension

### Manual Processing

```go
// Create dimension service
dimensionService := vision.NewDimensionService("", 0.5)

// Set calibration
calibration := config.GetCameraCalibration()
dimensionService.SetCalibration(calibration)

// Process image
dimensions, err := dimensionService.ProcessImage("path/to/image.jpg")

// Get vehicle classification
for _, dim := range dimensions {
    vehicleClass := dimensionService.ClassifyVehicle(dim)
    fmt.Printf("Class: %s (%.2f%%)\n", vehicleClass.Description, vehicleClass.Confidence*100)
}
```

## Database Schema

Service akan otomatis membuat table:

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

Dan menambah kolom ke `transact_anpr_capture`:

```sql
ALTER TABLE transact_anpr_capture
ADD COLUMN vehicle_length DECIMAL(10, 3),
ADD COLUMN vehicle_width DECIMAL(10, 3),
ADD COLUMN vehicle_height DECIMAL(10, 3),
ADD COLUMN vehicle_class VARCHAR(50),
ADD COLUMN dimension_confidence DECIMAL(5, 4);
```

## Vehicle Classification

Sistem mengklasifikasikan kendaraan berdasarkan dimensi:

| Class      | Length   | Width    | Description           |
| ---------- | -------- | -------- | --------------------- |
| motorcycle | < 2.5m   | < 1.5m   | Sepeda Motor          |
| sedan      | 2.5-5.5m | < 2.0m   | Mobil Penumpang       |
| suv        | 4.0-6.0m | 1.8-2.2m | SUV/Minivan           |
| truck      | 5.5-12m  | -        | Truk/Kendaraan Barang |
| bus        | > 7.0m   | > 2.0m   | Bus                   |

## Limitations

⚠️ **Catatan Penting:**

1. **Height Estimation**: Tinggi kendaraan adalah estimasi (tidak presisi) karena butuh side-view camera
2. **Accuracy**: Akurasi tergantung pada kualitas kalibrasi camera
3. **Distance**: Semakin jauh kendaraan dari camera, akurasi berkurang
4. **Angle**: Kendaraan harus terlihat dari angle yang baik (tidak terlalu miring)
5. **Mock Detection**: Saat ini menggunakan mock detection, untuk production perlu ML model (YOLO/OpenCV)

## Roadmap / Future Improvements

- [ ] Integrasi YOLO untuk real vehicle detection
- [ ] Support multiple camera views untuk height accuracy
- [ ] 3D reconstruction dari multiple angles
- [ ] Auto-calibration menggunakan known vehicle dimensions
- [ ] Real-time processing optimization
- [ ] Perspective transform untuk bird's eye view

## Testing

Untuk testing dengan gambar sample:

```bash
# Set DIMENSION_ENABLED=true
# Upload gambar ANPR Anda ke FTP
# Service akan otomatis memproses dan log hasil dimensi
# Check database untuk hasil yang tersimpan
```

## Support

Untuk hasil yang optimal, pastikan:

- ✅ Camera terpasang stabil (tidak bergerak)
- ✅ Kalibrasi dilakukan dengan teliti
- ✅ Reference object measured accurately
- ✅ Lighting condition consistent
- ✅ Camera lens tidak distorsi berlebihan
