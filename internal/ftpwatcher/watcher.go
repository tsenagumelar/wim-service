package ftpwatcher

import (
	"context"
	"log"
	"time"

	"github.com/jlaffaye/ftp"
)

// NewFileHandler dipanggil untuk setiap file yang ada di FTP.
// Handler bertanggung jawab menghapus file setelah selesai diproses.
type NewFileHandler func(ctx context.Context, c *ftp.ServerConn, name string) bool

type Watcher struct {
	Addr      string
	User      string
	Pass      string
	RemoteDir string
	Interval  time.Duration
	OnNewFile NewFileHandler
	conn      *ftp.ServerConn
}

func New(addr, user, pass, dir string, interval time.Duration, fn NewFileHandler) *Watcher {
	return &Watcher{
		Addr:      addr,
		User:      user,
		Pass:      pass,
		RemoteDir: dir,
		Interval:  interval,
		OnNewFile: fn,
	}
}

func (w *Watcher) connect() error {
	c, err := ftp.Dial(w.Addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return err
	}
	if err := c.Login(w.User, w.Pass); err != nil {
		return err
	}
	w.conn = c
	log.Println("[FTP] connected")
	return nil
}

func (w *Watcher) Start(ctx context.Context) error {
	if err := w.connect(); err != nil {
		return err
	}

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[FTP] stopped")
			return nil
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Watcher) poll(ctx context.Context) {
	entries, err := w.conn.List(w.RemoteDir)
	if err != nil {
		log.Println("[FTP] list error:", err)
		return
	}

	for _, e := range entries {
		if e.Type != ftp.EntryTypeFile {
			continue
		}

		log.Println("[FTP] file seen:", e.Name)

		if w.OnNewFile == nil {
			continue
		}

		// Handler yang akan memutuskan sukses/gagal.
		// Begitu sukses, handler akan menghapus file dari FTP,
		// sehingga di polling berikutnya file itu sudah tidak ada.
		w.OnNewFile(ctx, w.conn, e.Name)
	}
}
