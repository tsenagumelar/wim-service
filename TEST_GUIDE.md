# Test Vehicle Dimension Detection

## Cara Testing

### 1. Test dengan Gambar Sample

```bash
# Download atau copy gambar ANPR ke folder project
cp /path/ke/gambar/anpr.jpg ./test_image.jpg

# Jalankan test
go run test_dimension.go
```

### 2. Test dengan Service yang Berjalan

```bash
# Jalankan service
go run main.go

# Service akan otomatis memproses setiap gambar ANPR yang masuk dari FTP
# Check log untuk melihat hasil dimensi
```

### 3. Verifikasi Hasil di Database

```sql
-- Check hasil dimensi yang tersimpan
SELECT
    id,
    length_meters,
    width_meters,
    height_meters,
    vehicle_class,
    confidence,
    processed_at
FROM vehicle_dimensions
ORDER BY processed_at DESC
LIMIT 10;

-- Check ANPR dengan dimensi
SELECT
    plate_no,
    vehicle_length,
    vehicle_width,
    vehicle_height,
    vehicle_class,
    dimension_confidence,
    captured_at
FROM transact_anpr_capture
WHERE vehicle_length IS NOT NULL
ORDER BY captured_at DESC
LIMIT 10;
```

### 4. Test Kalibrasi

Untuk memverifikasi kalibrasi akurat:

1. **Pilih kendaraan yang sudah diketahui dimensinya**
   - Contoh: Toyota Avanza (P: 4.4m, L: 1.75m)
2. **Ambil foto dari camera ANPR**

3. **Jalankan test**

   ```bash
   go run test_dimension.go
   ```

4. **Bandingkan hasil dengan dimensi sebenarnya**
   - Jika selisih > 20%, kalibrasi perlu diperbaiki
   - Adjust parameter `CAMERA_REF_*` di .env

### 5. Test dengan Berbagai Jarak

```bash
# Test kendaraan di berbagai posisi:
# - Dekat camera (5-10m)
# - Sedang (10-20m)
# - Jauh (20-30m)

# Akurasi akan berkurang semakin jauh dari camera
```

### 6. Visualisasi Detection (Optional)

Edit `test_dimension.go` untuk save gambar dengan bounding box:

```go
// Tambahkan setelah ProcessImageFile
boxes, _ := dimensionHandler.DimensionService.Detector.DetectVehicle(testImagePath)
dimensionHandler.DimensionService.Detector.DrawBoundingBoxes(
    testImagePath,
    boxes,
    "./output_with_bbox.jpg",
)
log.Println("Saved image with bounding boxes: ./output_with_bbox.jpg")
```

## Expected Output

```
=== Vehicle Dimension Detection Test ===
[CONFIG] .env file loaded successfully
[CONFIG] Database connection established

Camera Calibration:
  Resolution: 1920x1080
  Focal Length: 1000.00 pixels
  Height: 6.00 m
  Tilt Angle: 30.00°
  Reference: 200 pixels = 5.00 m at 10.00 m distance
  Pixel-to-Meter Ratio: 0.025000 m/pixel

=== Processing Test Image: ./test_image.jpg ===
[DIMENSION] Processing image: ./test_image.jpg
[DIMENSION] Detected 1 vehicle(s)
[DIMENSION] Calculating dimensions for vehicle 1 (score: 0.95)
[DIMENSION] Vehicle 1 dimensions: L=4.52m W=1.85m H=1.11m (distance: 12.30m, confidence: 0.67)

=== Results ===
Success: true
Vehicle Count: 1

--- Vehicle 1 ---
Length: 4.52 meters
Width: 1.85 meters
Height: 1.11 meters (estimated)
Distance: 12.30 meters
Confidence: 67%
Classification: sedan (Mobil Penumpang / Sedan)

=== Test Complete ===
```

## Tips Kalibrasi

### Jika Hasil Tidak Akurat:

1. **Dimensi terlalu besar/kecil**

   - Adjust `CAMERA_REF_REAL_LENGTH` dan `CAMERA_REF_PIXEL_LENGTH`
   - Pastikan pengukuran reference object akurat

2. **Konsisten salah**

   - Check `CAMERA_HEIGHT_METERS` (mungkin salah ukur)
   - Check `CAMERA_TILT_ANGLE` (terlalu besar/kecil)

3. **Akurat di jarak dekat, salah di jauh**

   - Normal, gunakan rata-rata dari berbagai jarak
   - Atau gunakan multiple reference points

4. **Tinggi kendaraan tidak akurat**
   - Normal, karena perlu side-view camera
   - Height adalah estimasi (60% dari width)

## Quick Test Command

```bash
# Set DIMENSION_ENABLED=true di .env
# Copy test image
cp /path/to/anpr/image.jpg ./test_image.jpg

# Run test
go run test_dimension.go

# Check output dan verify dengan dimensi sebenarnya
```
