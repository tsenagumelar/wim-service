package handler

import (
	"context"
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
	RemoteDir string
	Minio     *minio.Client
	Bucket    string
}

func NewFileProcessor(remoteDir, endpoint, accessKey, secretKey, bucket string, useSSL bool) (*FileProcessor, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &FileProcessor{
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

	log.Println("[HANDLER] processing xml:", name)

	meta, err := p.parseXML(ctx, c, name)
	if err != nil {
		log.Println("[HANDLER] parse xml error:", err)
		// kalau XML corrupt, anggap selesai supaya tidak infinite retry
		return true
	}

	log.Printf("[HANDLER] plate=%s time=%s cam=%s conf=%s id=%s\n",
		meta.Plate, meta.FrameTime, meta.CameraID, meta.Confidence, meta.ID)

	// Format tanggal hari ini -> 03122025 (ddMMyyyy)
	datePrefix := time.Now().Format("02012006")

	// cari file jpg yang match dengan nama xml
	fullImg, plateImg, err := p.findImagesForXML(c, name)
	if err != nil {
		log.Println("[HANDLER] find images error:", err)
		// gambar belum siap -> nanti dicoba lagi
		return false
	}

	// Object name di MinIO: bucket/03122025/original-filename
	xmlObj := fmt.Sprintf("%s/%s", datePrefix, name)
	fullObj := fmt.Sprintf("%s/%s", datePrefix, fullImg)
	plateObj := fmt.Sprintf("%s/%s", datePrefix, plateImg)

	// upload XML
	if err := p.uploadXML(ctx, c, name, xmlObj); err != nil {
		log.Println("[HANDLER] upload xml error:", err)
		return false
	}

	// upload 2 image
	if err := p.uploadImage(ctx, c, fullImg, fullObj); err != nil {
		log.Println("[HANDLER] upload full img error:", err)
		return false
	}
	if err := p.uploadImage(ctx, c, plateImg, plateObj); err != nil {
		log.Println("[HANDLER] upload plate img error:", err)
		return false
	}

	// semua sukses -> hapus dari FTP
	if err := p.deleteFTP(c, []string{name, fullImg, plateImg}); err != nil {
		log.Println("[HANDLER] delete ftp error:", err)
		// di tahap ini file sudah ada di MinIO, boleh dianggap selesai
		return true
	}

	log.Println("[HANDLER] done id:", meta.ID)
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

	log.Println("[HANDLER] uploaded xml to minio:", objectName)
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
	log.Println("[HANDLER] uploaded image to minio:", objectName)
	return nil
}

func (p *FileProcessor) deleteFTP(c *ftp.ServerConn, names []string) error {
	for _, n := range names {
		fp := path.Join(p.RemoteDir, n)
		log.Println("[HANDLER] delete ftp:", fp)
		if err := c.Delete(fp); err != nil {
			return fmt.Errorf("delete %s: %w", fp, err)
		}
	}
	return nil
}
