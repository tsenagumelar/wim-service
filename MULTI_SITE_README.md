# WIM Service - Multi-Site Architecture

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture Concept](#architecture-concept)
- [Database Schema](#database-schema)
- [Implementation Guide](#implementation-guide)
- [Configuration](#configuration)
- [Deployment Scenarios](#deployment-scenarios)
- [Migration Guide](#migration-guide)
- [Best Practices](#best-practices)

---

## Overview

Multi-Site Architecture untuk WIM (Weigh-In-Motion) Service yang memungkinkan deployment di multiple lokasi dengan data centralized atau distributed.

### Key Features

- ✅ Multiple site support dengan identifikasi unik per lokasi
- ✅ Flexible deployment: local-only atau local + central synchronization
- ✅ Backward compatible dengan existing single-site deployment
- ✅ Easy redeployment ke server/lokasi baru
- ✅ Centralized reporting dan monitoring (optional)

### Use Cases

1. **Toll Road Network**: Multiple toll gates dengan centralized monitoring di HQ
2. **Regional WIM Stations**: Beberapa lokasi WIM di region berbeda
3. **Scalable Deployment**: Mudah add new site tanpa affect existing sites
4. **Disaster Recovery**: Each site independent, central sebagai backup

---

## Architecture Concept

### 1. Deployment Model

#### Model A: Local Only (Independent)

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Site 1 (JKT)  │     │   Site 2 (SBY)  │     │   Site 3 (BDG)  │
│                 │     │                 │     │                 │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │ WIM App   │  │     │  │ WIM App   │  │     │  │ WIM App   │  │
│  └─────┬─────┘  │     │  └─────┬─────┘  │     │  └─────┬─────┘  │
│        │        │     │        │        │     │        │        │
│  ┌─────▼─────┐  │     │  ┌─────▼─────┐  │     │  ┌─────▼─────┐  │
│  │ Local DB  │  │     │  │ Local DB  │  │     │  │ Local DB  │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

**Karakteristik:**

- Setiap site completely independent
- Tidak ada central database
- Cocok untuk: pilot project, testing, atau lokasi remote tanpa internet stable

#### Model B: Local + Central (Hybrid)

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Site 1 (JKT)  │     │   Site 2 (SBY)  │     │   Site 3 (BDG)  │
│                 │     │                 │     │                 │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │ WIM App   │  │     │  │ WIM App   │  │     │  │ WIM App   │  │
│  └─────┬─────┘  │     │  └─────┬─────┘  │     │  └─────┬─────┘  │
│        │        │     │        │        │     │        │        │
│  ┌─────▼─────┐  │     │  ┌─────▼─────┐  │     │  ┌─────▼─────┐  │
│  │ Local DB  │  │     │  │ Local DB  │  │     │  │ Local DB  │  │
│  └─────┬─────┘  │     │  └─────┬─────┘  │     │  └─────┬─────┘  │
└────────┼─────────┘     └────────┼─────────┘     └────────┼─────────┘
         │                        │                        │
         │ Sync (Optional)        │                        │
         └────────────────────────┼────────────────────────┘
                                  │
                          ┌───────▼────────┐
                          │   HQ/Central   │
                          │                │
                          │ ┌────────────┐ │
                          │ │Central DB  │ │
                          │ │(Aggregate) │ │
                          │ └────────────┘ │
                          └────────────────┘
```

**Karakteristik:**

- Setiap site punya local database (tetap berfungsi walau central down)
- Optional sync ke central untuk reporting/monitoring
- Central database aggregate data dari semua site
- Cocok untuk: production environment dengan centralized monitoring

### 2. Site Identification

Setiap site memiliki identifikasi unik:

- **SITE_ID**: Code unik (e.g., `JKT-TOLL-01`, `SBY-WIM-01`)
- **SITE_NAME**: Nama site yang human-readable
- **SITE_LOCATION**: Alamat/lokasi fisik
- **SITE_REGION**: Region/area untuk grouping

Semua transaksi (ANPR capture, axle data) akan di-tag dengan `site_id`.

---

## Database Schema

### Schema Structure

Database schema mengikuti pattern dari **`migrations/ddl.sql`**. Untuk multi-site, ditambahkan:

#### 1. Master Site Table

Lihat: `migrations/100_simple_multi_site.sql`

```sql
CREATE TABLE public.master_site (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    code varchar(50) UNIQUE NOT NULL,
    site_name varchar(200) NOT NULL,
    site_location varchar(200),
    site_region varchar(100),
    description text,
    is_active bool DEFAULT true,
    is_deleted bool DEFAULT false,
    created_by uuid,
    created_date timestamptz DEFAULT now(),
    updated_by uuid,
    updated_date timestamptz DEFAULT now()
);
```

#### 2. Transaction Tables dengan site_id

Existing transaction tables di **`migrations/ddl.sql`** ditambahkan kolom `site_id`:

**transact_anpr_capture:**

```sql
ALTER TABLE public.transact_anpr_capture
ADD COLUMN site_id uuid REFERENCES public.master_site(id);
```

**transact_axle_capture:**

```sql
ALTER TABLE public.transact_axle_capture
ADD COLUMN site_id uuid REFERENCES public.master_site(id);
```

### Complete Schema Reference

| Table                   | Purpose                   | Schema Reference                  |
| ----------------------- | ------------------------- | --------------------------------- |
| `master_device_type`    | Device types              | `migrations/ddl.sql`              |
| `master_role`           | User roles                | `migrations/ddl.sql`              |
| `master_vehicle_class`  | Vehicle classifications   | `migrations/ddl.sql`              |
| `master_config`         | System configurations     | `migrations/ddl.sql`              |
| `master_device`         | Device registry           | `migrations/ddl.sql`              |
| `master_user`           | User accounts             | `migrations/ddl.sql`              |
| `transact_anpr_capture` | ANPR captures + `site_id` | `migrations/ddl.sql` + multi-site |
| `transact_axle_capture` | Axle data + `site_id`     | `migrations/ddl.sql` + multi-site |
| `user_login_history`    | Login tracking            | `migrations/ddl.sql`              |

**Important:**

- Semua table structure mengikuti pattern di `ddl.sql`
- Multi-site hanya menambah `master_site` dan kolom `site_id` di transaction tables
- Naming convention: `master_*` untuk master data, `transact_*` untuk transactions
- Standard audit fields: `is_active`, `is_deleted`, `created_by/date`, `updated_by/date`

---

## Implementation Guide

### Step 1: Setup Database

#### For New Database (Fresh Install)

```bash
# 1. Create database
createdb wim_site_jkt

# 2. Run main schema
psql -d wim_site_jkt -f migrations/ddl.sql

# 3. Run multi-site schema
psql -d wim_site_jkt -f migrations/100_simple_multi_site.sql
```

#### For Existing Database (Migration)

```bash
# Run multi-site migration only
psql -d existing_wim_db -f migrations/100_simple_multi_site.sql
```

Script ini akan:

- ✅ Create `master_site` table
- ✅ Add `site_id` column ke transaction tables (jika belum ada)
- ✅ Set default site untuk existing data
- ✅ Create foreign key constraints

### Step 2: Configure Application

Edit `.env` file untuk setiap site:

```env
# Site Identification
SITE_ID=JKT-TOLL-01
SITE_NAME=Jakarta Toll Gate 1
SITE_LOCATION=Jakarta Outer Ring Road KM 12
SITE_REGION=Jakarta

# Local Database (Required)
DATABASE_URL=postgres://user:pass@localhost:5432/wim_site_jkt

# Central Database (Optional - for sync)
CENTRAL_DATABASE_URL=postgres://user:pass@central.example.com:5432/wim_central
SYNC_ENABLED=true

# Other configs...
FTP_HOST=ftp.local.site
FTP_PORT=21
```

### Step 3: Update Application Code

Kode sudah di-update untuk support multi-site:

**File Modified:**

- `internal/config/config.go` - Added site configuration fields
- `internal/handler/anpr_handler.go` - Added `site_id` to ANPR records
- `internal/handler/dimension_handler.go` - Added `site_id` to dimension records
- `main.go` - Pass `SiteID` to handlers

**Verification:**

```bash
# Build
go build -o wim-service

# Run
./wim-service
```

Log akan menampilkan:

```
INFO: Starting WIM Service
INFO: Site: JKT-TOLL-01 (Jakarta Toll Gate 1)
INFO: Location: Jakarta Outer Ring Road KM 12
INFO: Region: Jakarta
INFO: Sync to Central: enabled
```

---

## Configuration

### Environment Variables

| Variable               | Required | Default | Description               |
| ---------------------- | -------- | ------- | ------------------------- |
| `SITE_ID`              | ✅ Yes   | -       | Unique site code          |
| `SITE_NAME`            | ✅ Yes   | -       | Site display name         |
| `SITE_LOCATION`        | ❌ No    | -       | Physical location         |
| `SITE_REGION`          | ❌ No    | -       | Region/area grouping      |
| `DATABASE_URL`         | ✅ Yes   | -       | Local database connection |
| `CENTRAL_DATABASE_URL` | ❌ No    | -       | Central DB for sync       |
| `SYNC_ENABLED`         | ❌ No    | `false` | Enable/disable sync       |

### Site Code Convention

Recommended naming pattern untuk `SITE_ID`:

```
[LOCATION]-[TYPE]-[NUMBER]

Examples:
- JKT-TOLL-01    (Jakarta, Toll Gate, Station 1)
- SBY-WIM-01     (Surabaya, WIM Station, Station 1)
- BDG-TOLL-02    (Bandung, Toll Gate, Station 2)
- MDN-WIM-MAIN   (Medan, WIM, Main Station)
```

---

## Deployment Scenarios

### Scenario 1: Deploy Site Baru (Independent)

Deploy WIM service di lokasi baru tanpa central connection:

```bash
# 1. Setup server baru
ssh user@new-site-server

# 2. Install dependencies
sudo apt install postgresql-14 golang-1.21

# 3. Create database
createdb wim_new_site

# 4. Run migrations
psql -d wim_new_site -f migrations/ddl.sql
psql -d wim_new_site -f migrations/100_simple_multi_site.sql

# 5. Configure site
cat > .env << EOF
SITE_ID=NEW-SITE-01
SITE_NAME=New Site Station 1
SITE_LOCATION=New Location Address
SITE_REGION=NewRegion
DATABASE_URL=postgres://user:pass@localhost:5432/wim_new_site
SYNC_ENABLED=false
EOF

# 6. Deploy & run
go build -o wim-service
./wim-service
```

### Scenario 2: Connect Existing Site ke Central

Menambah central database untuk existing site:

```bash
# 1. Edit .env - add central DB config
CENTRAL_DATABASE_URL=postgres://user:pass@central.example.com:5432/wim_central
SYNC_ENABLED=true

# 2. Restart service
systemctl restart wim-service

# 3. Verify sync
# Check logs untuk sync messages
journalctl -u wim-service -f
```

### Scenario 3: Setup Central Database

Setup central database untuk aggregate data dari multiple sites:

```bash
# 1. Create central database
createdb wim_central

# 2. Run same migrations
psql -d wim_central -f migrations/ddl.sql
psql -d wim_central -f migrations/100_simple_multi_site.sql

# 3. Insert all sites info
psql -d wim_central << EOF
INSERT INTO public.master_site (code, site_name, site_location, site_region) VALUES
    ('JKT-TOLL-01', 'Jakarta Toll Gate 1', 'Jakarta JORR KM 12', 'Jakarta'),
    ('SBY-WIM-01', 'Surabaya WIM 1', 'Surabaya-Gempol KM 5', 'Surabaya'),
    ('BDG-TOLL-01', 'Bandung Toll Gate 1', 'Bandung Ring Road KM 8', 'Bandung');
EOF

# 4. Central database siap receive data from all sites
```

### Scenario 4: Multi-Region Deployment

Deploy di multiple regions dengan regional central:

```
Region Jakarta:           Region Surabaya:           HQ Central:
- JKT-TOLL-01            - SBY-WIM-01               - Aggregate all
- JKT-TOLL-02            - SBY-WIM-02               - Reporting
- JKT-WIM-01             - SBY-TOLL-01              - Monitoring
    ↓                         ↓                           ↑
[Regional DB JKT] ──────────────────────────────────────→ [HQ Central DB]
                              ↑
                    [Regional DB SBY]
```

---

## Migration Guide

### Migrate dari Single-Site ke Multi-Site

Jika sudah running single-site WIM service:

#### Step 1: Backup Database

```bash
pg_dump existing_db > backup_before_multisite.sql
```

#### Step 2: Run Multi-Site Migration

```bash
psql -d existing_db -f migrations/100_simple_multi_site.sql
```

Migration script akan:

- ✅ Create `master_site` table
- ✅ Add `site_id` column (if not exists)
- ✅ Insert default site (`SITE001`)
- ✅ Update existing records dengan default site
- ✅ Add foreign key constraints

#### Step 3: Verify Data

```sql
-- Check master_site
SELECT * FROM master_site;

-- Check existing data sudah punya site_id
SELECT COUNT(*), site_id
FROM transact_anpr_capture
GROUP BY site_id;

SELECT COUNT(*), site_id
FROM transact_axle_capture
GROUP BY site_id;
```

#### Step 4: Update .env

```env
# Add site identification
SITE_ID=SITE001
SITE_NAME=Existing Site
SITE_LOCATION=Current Location
SITE_REGION=Default

# Existing configs stay the same
DATABASE_URL=postgres://user:pass@localhost:5432/existing_db
```

#### Step 5: Restart Application

```bash
# Rebuild with updated code
go build -o wim-service

# Restart service
./wim-service
```

---

## Best Practices

### 1. Database Management

**✅ DO:**

- Backup database regularly di setiap site
- Use connection pooling untuk efficiency
- Monitor disk space di local DB
- Set retention policy untuk old data

**❌ DON'T:**

- Jangan delete `master_site` record yang masih referenced
- Jangan manually edit `site_id` di transaction tables
- Jangan share same database untuk multiple sites

### 2. Site Configuration

**✅ DO:**

- Use meaningful `SITE_ID` codes (location-based)
- Document setiap site configuration
- Keep `.env` file secure (don't commit to git)
- Use environment-specific configs

**❌ DON'T:**

- Jangan use duplicate `SITE_ID`
- Jangan hardcode site config di application code
- Jangan expose sensitive config di logs

### 3. Deployment

**✅ DO:**

- Test migration di staging environment first
- Verify data integrity after migration
- Monitor application logs after deployment
- Keep migration scripts versioned

**❌ DON'T:**

- Jangan run migration di production tanpa testing
- Jangan skip backup sebelum migration
- Jangan force deployment jika ada errors

### 4. Synchronization (if enabled)

**✅ DO:**

- Implement retry logic untuk sync failures
- Log sync activities
- Monitor sync lag/delay
- Handle network interruptions gracefully

**❌ DON'T:**

- Jangan block local operations saat sync
- Jangan assume central always available
- Jangan sync sensitive data tanpa encryption

### 5. Monitoring & Maintenance

**✅ DO:**

- Monitor database size di setiap site
- Track transaction volume per site
- Alert on sync failures
- Regular health checks

**❌ DON'T:**

- Jangan ignore warning logs
- Jangan let database grow unbounded
- Jangan skip regular maintenance

---

## Troubleshooting

### Problem: Site ID not found in database

**Symptom:**

```
ERROR: Site SITE001 not found in master_site table
```

**Solution:**

```sql
-- Insert missing site
INSERT INTO public.master_site (code, site_name, site_region, is_active)
VALUES ('SITE001', 'Default Site', 'Default', true);
```

### Problem: Foreign key constraint violation

**Symptom:**

```
ERROR: insert or update on table "transact_anpr_capture" violates foreign key constraint "fk_anpr_site"
```

**Solution:**

```sql
-- Check if site_id exists in master_site
SELECT id, code FROM master_site WHERE code = 'YOUR-SITE-ID';

-- If not exists, insert it first
-- Then retry the transaction
```

### Problem: Existing data has NULL site_id

**Solution:**

```sql
-- Update NULL site_id to default site
UPDATE transact_anpr_capture
SET site_id = (SELECT id FROM master_site WHERE code = 'SITE001')
WHERE site_id IS NULL;

UPDATE transact_axle_capture
SET site_id = (SELECT id FROM master_site WHERE code = 'SITE001')
WHERE site_id IS NULL;
```

---

## Query Examples

### Get capture count per site

```sql
SELECT
    s.code,
    s.site_name,
    COUNT(t.id) as total_captures
FROM master_site s
LEFT JOIN transact_anpr_capture t ON s.id = t.site_id
GROUP BY s.code, s.site_name
ORDER BY total_captures DESC;
```

### Get recent captures from specific site

```sql
SELECT
    t.*,
    s.site_name
FROM transact_anpr_capture t
JOIN master_site s ON t.site_id = s.id
WHERE s.code = 'JKT-TOLL-01'
ORDER BY t.captured_at DESC
LIMIT 100;
```

### Compare vehicle class distribution across sites

```sql
SELECT
    s.code as site_code,
    vc.type as vehicle_class,
    COUNT(*) as count
FROM transact_axle_capture t
JOIN master_site s ON t.site_id = s.id
JOIN master_vehicle_class vc ON t.vehicle_category = vc.type
GROUP BY s.code, vc.type
ORDER BY s.code, count DESC;
```

---

## Support & Contact

Untuk pertanyaan atau issue terkait multi-site architecture:

1. Check dokumentasi ini dan `migrations/ddl.sql` untuk reference
2. Review migration script di `migrations/100_simple_multi_site.sql`
3. Check application logs untuk detailed error messages
4. Contact development team jika perlu assistance

---

**Last Updated:** December 10, 2025  
**Version:** 1.0  
**Schema Reference:** `migrations/ddl.sql` + `migrations/100_simple_multi_site.sql`
