package handler

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ANPRMetadata struct {
	Plate      string
	FrameTime  string
	Location   string
	CameraID   string
	Confidence string
	ID         string
}

// Sesuaikan dengan struktur XML dari kamera
type xmlResult struct {
	Location struct {
		Value string `xml:"value,attr"`
	} `xml:"location"`

	CameraID struct {
		Value string `xml:"value,attr"`
	} `xml:"cameraid"`

	ID struct {
		Value string `xml:"value,attr"`
	} `xml:"ID"`

	Capture struct {
		FrameTime struct {
			Value string `xml:"value,attr"`
		} `xml:"frametime"`
	} `xml:"capture"`

	ANPR struct {
		Text struct {
			Value string `xml:"value,attr"`
		} `xml:"text"`
		Confidence struct {
			Value string `xml:"value,attr"`
		} `xml:"confidence"`
	} `xml:"anpr"`
}

type FileProcessor struct {
	DB               *sql.DB
	SiteID           string // Site identifier for multi-site deployment
	RemoteDir        string
	Minio            *minio.Client
	Bucket           string
	DimensionHandler *DimensionHandler // Optional: for vehicle dimension detection
}

// SetDimensionHandler sets the dimension handler for processing vehicle dimensions
func (p *FileProcessor) SetDimensionHandler(handler *DimensionHandler) {
	p.DimensionHandler = handler
}

func NewFileProcessor(db *sql.DB, siteID, remoteDir, endpoint, accessKey, secretKey, bucket string, useSSL bool) (*FileProcessor, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &FileProcessor{
		DB:        db,
		SiteID:    siteID,
		RemoteDir: remoteDir,
		Minio:     mc,
		Bucket:    bucket,
	}, nil
}

// HandleNewFile dipanggil setiap ada file di FTP.
// Kita hanya proses XML; JPG akan dicari berdasarkan nama XML-nya.
func (p *FileProcessor) HandleNewFile(ctx context.Context, c *ftp.ServerConn, name string) bool {
	// hanya proses XML
	if !strings.HasSuffix(strings.ToLower(name), ".xml") {
		return true
	}

	log.Println("[ANPR] processing xml:", name)

	meta, err := p.parseXML(ctx, c, name)
	if err != nil {
		log.Println("[ANPR] parse xml error:", err)
		// kalau XML corrupt, anggap selesai supaya tidak infinite retry
		return true
	}

	log.Printf("[ANPR] plate=%s time=%s cam=%s conf=%s id=%s\n",
		meta.Plate, meta.FrameTime, meta.CameraID, meta.Confidence, meta.ID)

	// Format tanggal hari ini -> 03122025 (ddMMyyyy)
	datePrefix := time.Now().Format("02012006")

	// cari file jpg yang match dengan nama xml
	fullImg, plateImg, err := p.findImagesForXML(c, name)
	if err != nil {
		log.Println("[ANPR] find images error:", err)
		// gambar belum siap -> nanti dicoba lagi
		return false
	}

	// Object name di MinIO: bucket/03122025/original-filename
	xmlObj := fmt.Sprintf("%s/%s", datePrefix, name)
	fullObj := fmt.Sprintf("%s/%s", datePrefix, fullImg)
	plateObj := fmt.Sprintf("%s/%s", datePrefix, plateImg)

	// upload XML
	if err := p.uploadXML(ctx, c, name, xmlObj); err != nil {
		log.Println("[ANPR] upload xml error:", err)
		return false
	}

	// upload 2 image
	if err := p.uploadImage(ctx, c, fullImg, fullObj); err != nil {
		log.Println("[ANPR] upload full img error:", err)
		return false
	}
	if err := p.uploadImage(ctx, c, plateImg, plateObj); err != nil {
		log.Println("[ANPR] upload plate img error:", err)
		return false
	}

	// insert ke database
	if err := p.insertANPRRecord(ctx, meta, datePrefix, xmlObj, fullObj, plateObj); err != nil {
		log.Println("[ANPR] insert DB error:", err)
		// gagal insert -> jangan hapus dari FTP supaya bisa diproses ulang
		return false
	}

	// Process vehicle dimensions if handler is set
	if p.DimensionHandler != nil {
		log.Printf("[ANPR] Processing vehicle dimensions for plate: %s", meta.Plate)
		// Download full image temporarily for dimension processing
		// In production, you might want to download from MinIO or keep FTP file temporarily
		if err := p.processDimensions(ctx, meta, fullObj); err != nil {
			log.Printf("[ANPR] Warning: Failed to process dimensions: %v", err)
			// Don't fail the whole process if dimension detection fails
		}
	}

	// semua sukses -> hapus dari FTP
	if err := p.deleteFTP(c, []string{name, fullImg, plateImg}); err != nil {
		log.Println("[ANPR] delete ftp error:", err)
		// di tahap ini file sudah ada di MinIO, boleh dianggap selesai
		return true
	}

	log.Println("[ANPR] done id:", meta.ID)
	return true
}

func (p *FileProcessor) parseXML(ctx context.Context, c *ftp.ServerConn, name string) (*ANPRMetadata, error) {
	r, err := c.Retr(path.Join(p.RemoteDir, name))
	if err != nil {
		return nil, fmt.Errorf("ftp retr xml: %w", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read xml: %w", err)
	}

	var x xmlResult
	if err := xml.Unmarshal(b, &x); err != nil {
		return nil, fmt.Errorf("unmarshal xml: %w", err)
	}

	return &ANPRMetadata{
		Plate:      x.ANPR.Text.Value,
		FrameTime:  x.Capture.FrameTime.Value,
		Location:   x.Location.Value,
		CameraID:   x.CameraID.Value,
		Confidence: x.ANPR.Confidence.Value,
		ID:         x.ID.Value,
	}, nil
}

// Cari 2 file JPG yang prefix-nya sama dengan nama XML
// contoh:
//
//	xml:     1764569194214.xml
//	full:    1764569194214.xml.jpeg
//	plate:   1764569194214.xml.plate.jpg
func (p *FileProcessor) findImagesForXML(c *ftp.ServerConn, xmlName string) (fullImg, plateImg string, err error) {
	entries, err := c.List(p.RemoteDir)
	if err != nil {
		return "", "", fmt.Errorf("list dir: %w", err)
	}

	prefix := xmlName

	for _, e := range entries {
		if e.Type != ftp.EntryTypeFile {
			continue
		}
		if !strings.HasPrefix(e.Name, prefix) {
			continue
		}

		lower := strings.ToLower(e.Name)
		if strings.Contains(lower, "plate") &&
			(strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg")) {
			plateImg = e.Name
		} else if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") {
			// diasumsikan jpg lain adalah full image
			fullImg = e.Name
		}
	}

	if fullImg == "" || plateImg == "" {
		return "", "", fmt.Errorf("images not ready yet (full=%q plate=%q)", fullImg, plateImg)
	}

	return fullImg, plateImg, nil
}

func (p *FileProcessor) uploadXML(ctx context.Context, c *ftp.ServerConn, xmlName, objectName string) error {
	r, err := c.Retr(path.Join(p.RemoteDir, xmlName))
	if err != nil {
		return fmt.Errorf("ftp retr xml: %w", err)
	}
	defer r.Close()

	_, err = p.Minio.PutObject(ctx, p.Bucket, objectName, r, -1, minio.PutObjectOptions{
		ContentType: "application/xml",
	})
	if err != nil {
		return fmt.Errorf("minio put xml: %w", err)
	}

	log.Println("[ANPR] uploaded xml to minio:", objectName)
	return nil
}

func (p *FileProcessor) uploadImage(ctx context.Context, c *ftp.ServerConn, ftpName, objectName string) error {
	r, err := c.Retr(path.Join(p.RemoteDir, ftpName))
	if err != nil {
		return fmt.Errorf("ftp retr image: %w", err)
	}
	defer r.Close()

	_, err = p.Minio.PutObject(ctx, p.Bucket, objectName, r, -1, minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return fmt.Errorf("minio put image: %w", err)
	}
	log.Println("[ANPR] uploaded image to minio:", objectName)
	return nil
}

func (p *FileProcessor) deleteFTP(c *ftp.ServerConn, names []string) error {
	for _, n := range names {
		fp := path.Join(p.RemoteDir, n)
		log.Println("[ANPR] delete ftp:", fp)
		if err := c.Delete(fp); err != nil {
			return fmt.Errorf("delete %s: %w", fp, err)
		}
	}
	return nil
}

func (p *FileProcessor) insertANPRRecord(ctx context.Context, meta *ANPRMetadata, dateFolder, xmlObj, fullObj, plateObj string) error {

	// parse confidence (string -> float)
	var conf sql.NullFloat64
	if meta.Confidence != "" {
		if f, err := strconv.ParseFloat(meta.Confidence, 64); err == nil {
			conf.Valid = true
			conf.Float64 = f
		}
	}

	// parse frametime ke timestamptz (layout sesuai XML: 2025.12.01 14:06:27.946)
	var capturedAt sql.NullTime
	if meta.FrameTime != "" {
		if t, err := time.Parse("2006.01.02 15:04:05.000", meta.FrameTime); err == nil {
			capturedAt.Valid = true
			capturedAt.Time = t
		}
	}

	query := `
	INSERT INTO public.transact_anpr_capture
		(site_id, external_id, plate_no, confidence, captured_at,
		 location_code, camera_id,
		 minio_bucket, minio_date_folder,
		 minio_xml_object, minio_full_image_object, minio_plate_image_object,
		 synced_to_central)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,false)
	ON CONFLICT (external_id) DO UPDATE SET
		site_id = EXCLUDED.site_id,
		plate_no = EXCLUDED.plate_no,
		confidence = EXCLUDED.confidence,
		captured_at = EXCLUDED.captured_at,
		location_code = EXCLUDED.location_code,
		camera_id = EXCLUDED.camera_id,
		minio_bucket = EXCLUDED.minio_bucket,
		minio_date_folder = EXCLUDED.minio_date_folder,
		minio_xml_object = EXCLUDED.minio_xml_object,
		minio_full_image_object = EXCLUDED.minio_full_image_object,
		minio_plate_image_object = EXCLUDED.minio_plate_image_object,
		updated_date = now();
	`

	_, err := p.DB.ExecContext(
		ctx,
		query,
		p.SiteID, // NEW: Site identifier
		meta.ID,
		meta.Plate,
		conf,
		capturedAt,
		meta.Location,
		meta.CameraID,
		p.Bucket,
		dateFolder,
		xmlObj,
		fullObj,
		plateObj,
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	return nil
}

// processDimensions downloads the image from MinIO and processes vehicle dimensions
func (p *FileProcessor) processDimensions(ctx context.Context, meta *ANPRMetadata, objectName string) error {
	// Download image from MinIO to temporary file
	tmpFile := fmt.Sprintf("/tmp/anpr_%s.jpg", meta.ID)

	obj, err := p.Minio.GetObject(ctx, p.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("get object from minio: %w", err)
	}
	defer obj.Close()

	// Save to temporary file
	outFile, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, obj); err != nil {
		return fmt.Errorf("copy to temp file: %w", err)
	}

	// Process dimensions
	result, err := p.DimensionHandler.ProcessANPRImage(tmpFile, meta.Plate, meta.ID)
	if err != nil {
		return fmt.Errorf("process dimensions: %w", err)
	}

	log.Printf("[ANPR] Dimension processing complete: found %d vehicle(s)", result.VehicleCount)
	for i, dims := range result.Dimensions {
		log.Printf("[ANPR] Vehicle %d: L=%.2fm W=%.2fm H=%.2fm (confidence: %.2f)",
			i+1, dims.LengthMeters, dims.WidthMeters, dims.HeightMeters, dims.Confidence)
	}

	// Clean up temp file
	os.Remove(tmpFile)

	return nil
}
