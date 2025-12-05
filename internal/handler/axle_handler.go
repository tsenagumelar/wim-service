package handler

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ===== Metadata axle yg kita ambil =====

type AxleMetadata struct {
	Plate     string
	FrameTime string
	CameraID  string
	ID        string

	Length   int // mm
	NWheels  int
	NAxles   int
	Category string
	BodyType string
}

// Struktur XML untuk parsing axle
type axleXML struct {
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
	} `xml:"anpr"`

	VAC struct {
		Vehicle0 struct {
			Length struct {
				Value string `xml:"value,attr"`
			} `xml:"length"`
			NWheels struct {
				Value string `xml:"value,attr"`
			} `xml:"nwheels"`
			NAxles struct {
				Value string `xml:"value,attr"`
			} `xml:"naxles"`
			Category struct {
				Value string `xml:"value,attr"`
			} `xml:"category"`
			BodyType struct {
				Value string `xml:"value,attr"`
			} `xml:"body_type"`
		} `xml:"vehicle0"`
	} `xml:"vac"`
}

// ===== Processor untuk folder AXLE =====

type AxleProcessor struct {
	RemoteDir string
	Minio     *minio.Client
	Bucket    string
}

func NewAxleProcessor(remoteDir, endpoint, accessKey, secretKey, bucket string, useSSL bool) (*AxleProcessor, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &AxleProcessor{
		RemoteDir: remoteDir,
		Minio:     mc,
		Bucket:    bucket,
	}, nil
}

// Dipanggil watcher tiap kali ada file di folder AXLE
// Kita hanya proses file .xml
func (p *AxleProcessor) HandleNewFileAXLE(ctx context.Context, c *ftp.ServerConn, name string) bool {
	if !strings.HasSuffix(strings.ToLower(name), ".xml") {
		return true
	}

	log.Println("[AXLE] processing xml:", name)

	meta, err := p.parseAxleXML(ctx, c, name)
	if err != nil {
		log.Println("[AXLE] parse xml error:", err)
		// xml rusak → tandai selesai saja (tidak retry terus)
		return true
	}

	log.Printf("[AXLE] ID=%s Plate=%s Time=%s Cam=%s Length=%dmm Axles=%d Wheels=%d Cat=%s Body=%s\n",
		meta.ID, meta.Plate, meta.FrameTime, meta.CameraID,
		meta.Length, meta.NAxles, meta.NWheels, meta.Category, meta.BodyType)

	// Folder tanggal hari ini, misal: 03122025
	datePrefix := time.Now().Format("02012006")

	// cari 1 file jpg yg prefix-nya sama dengan nama xml
	imgName, err := p.findImageForAxleXML(c, name)
	if err != nil {
		log.Println("[AXLE] find image error:", err)
		// jpg belum ada → biarkan watcher retry di polling berikutnya
		return false
	}

	xmlObj := fmt.Sprintf("%s/%s", datePrefix, name)
	imgObj := fmt.Sprintf("%s/%s", datePrefix, imgName)

	if err := p.uploadXML(ctx, c, name, xmlObj); err != nil {
		log.Println("[AXLE] upload xml error:", err)
		return false
	}
	if err := p.uploadImage(ctx, c, imgName, imgObj); err != nil {
		log.Println("[AXLE] upload image error:", err)
		return false
	}

	if err := p.insertAxleRecord(ctx, meta, datePrefix, xmlObj, imgObj); err != nil {
		log.Println("[AXLE] insert DB error:", err)
		return false
	}

	// semua sudah ke-upload → hapus dari FTP
	if err := p.deleteFTP(c, []string{name, imgName}); err != nil {
		log.Println("[AXLE] delete ftp error:", err)
		// file sudah aman di MinIO, jadi anggap selesai
		return true
	}

	log.Println("[AXLE] done ID:", meta.ID)
	return true
}

func (p *AxleProcessor) parseAxleXML(ctx context.Context, c *ftp.ServerConn, name string) (*AxleMetadata, error) {
	r, err := c.Retr(path.Join(p.RemoteDir, name))
	if err != nil {
		return nil, fmt.Errorf("ftp retr xml: %w", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read xml: %w", err)
	}

	var x axleXML
	if err := xml.Unmarshal(b, &x); err != nil {
		return nil, fmt.Errorf("unmarshal xml: %w", err)
	}

	meta := &AxleMetadata{
		Plate:     x.ANPR.Text.Value,
		FrameTime: x.Capture.FrameTime.Value,
		CameraID:  x.CameraID.Value,
		ID:        x.ID.Value,
		Category:  x.VAC.Vehicle0.Category.Value,
		BodyType:  x.VAC.Vehicle0.BodyType.Value,
	}

	fmt.Sscanf(x.VAC.Vehicle0.Length.Value, "%d", &meta.Length)
	fmt.Sscanf(x.VAC.Vehicle0.NWheels.Value, "%d", &meta.NWheels)
	fmt.Sscanf(x.VAC.Vehicle0.NAxles.Value, "%d", &meta.NAxles)

	return meta, nil
}

func (p *AxleProcessor) findImageForAxleXML(c *ftp.ServerConn, xmlName string) (string, error) {
	entries, err := c.List(p.RemoteDir)
	if err != nil {
		return "", fmt.Errorf("list dir: %w", err)
	}

	prefix := xmlName // contoh: 1764570627075.xml
	var candidate string

	for _, e := range entries {
		if e.Type != ftp.EntryTypeFile {
			continue
		}
		if !strings.HasPrefix(e.Name, prefix) {
			continue
		}
		lower := strings.ToLower(e.Name)
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") {
			candidate = e.Name
			break
		}
	}

	if candidate == "" {
		return "", fmt.Errorf("image not found for xml %s", xmlName)
	}
	return candidate, nil
}

func (p *AxleProcessor) uploadXML(ctx context.Context, c *ftp.ServerConn, xmlName, objectName string) error {
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
	log.Println("[AXLE] uploaded xml to minio:", objectName)
	return nil
}

func (p *AxleProcessor) uploadImage(ctx context.Context, c *ftp.ServerConn, ftpName, objectName string) error {
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
	log.Println("[AXLE] uploaded image to minio:", objectName)
	return nil
}

func (p *AxleProcessor) deleteFTP(c *ftp.ServerConn, names []string) error {
	for _, n := range names {
		fp := path.Join(p.RemoteDir, n)
		log.Println("[AXLE] delete ftp:", fp)
		if err := c.Delete(fp); err != nil {
			return fmt.Errorf("delete %s: %w", fp, err)
		}
	}
	return nil
}

func (p *AxleProcessor) insertAxleRecord(ctx context.Context, meta *AxleMetadata, dateFolder, xmlObj, imgObj string) error {
	dbConn, err := getDB()
	if err != nil {
		return fmt.Errorf("getDB: %w", err)
	}

	var capturedAt sql.NullTime
	if meta.FrameTime != "" {
		if t, err := time.Parse("2006.01.02 15:04:05.000", meta.FrameTime); err == nil {
			capturedAt.Valid = true
			capturedAt.Time = t
		}
	}

	query := `
      INSERT INTO public.transact_axle_capture
      (external_id, plate_no, captured_at, camera_id,
       length_mm, total_wheels, total_axles, vehicle_category, vehicle_body_type,
       minio_bucket, minio_date_folder, minio_xml_object, minio_image_object)
      VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
      ON CONFLICT (external_id) DO UPDATE SET
       plate_no = EXCLUDED.plate_no,
       captured_at = EXCLUDED.captured_at,
       camera_id = EXCLUDED.camera_id,
       length_mm = EXCLUDED.length_mm,
       total_wheels = EXCLUDED.total_wheels,
       total_axles = EXCLUDED.total_axles,
       vehicle_category = EXCLUDED.vehicle_category,
       vehicle_body_type = EXCLUDED.vehicle_body_type,
       minio_bucket = EXCLUDED.minio_bucket,
       minio_date_folder = EXCLUDED.minio_date_folder,
       minio_xml_object = EXCLUDED.minio_xml_object,
       minio_image_object = EXCLUDED.minio_image_object,
       updated_date = now();
      `

	_, err = dbConn.ExecContext(
		ctx,
		query,
		meta.ID,
		meta.Plate,
		capturedAt,
		meta.CameraID,
		meta.Length,
		meta.NWheels,
		meta.NAxles,
		meta.Category,
		meta.BodyType,
		p.Bucket,
		dateFolder,
		xmlObj,
		imgObj,
	)
	if err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}
	return nil
}
