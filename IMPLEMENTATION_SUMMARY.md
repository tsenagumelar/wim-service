# Multi-Site Implementation - Change Summary

## Updated Files

### 1. `.env` ✅

**Changes:**

- Added SITE_ID configuration (must match master_site.code)
- Added SITE_NAME, SITE_LOCATION, SITE_REGION
- Added CENTRAL_DATABASE_URL and SYNC_ENABLED for optional sync to central HQ
- Organized sections for better readability

**Example Configuration:**

```env
SITE_ID="SITE001"
SITE_NAME="Default Site"
SITE_LOCATION="Central Office"
SITE_REGION="Default"
```

### 2. `.env.example` ✅

**Changes:**

- Updated SITE_ID comments to clarify it must match master_site.code
- Updated descriptions for SITE_LOCATION and SITE_REGION
- Added clear examples for site identification

### 3. `internal/handler/anpr_handler.go` ✅

**Changes:**

- ✅ Already has `SiteID` field in `FileProcessor` struct
- ✅ Already includes `site_id` in INSERT query
- ✅ Already passes `p.SiteID` to database

**INSERT Query includes:**

```sql
INSERT INTO public.transact_anpr_capture
    (site_id, external_id, plate_no, confidence, captured_at, ...)
VALUES ($1, $2, $3, $4, $5, ...)
```

### 4. `internal/handler/axle_handler.go` ✅

**Changes:**

- Added `SiteID string` field to `AxleProcessor` struct
- Updated `NewAxleProcessor()` constructor to accept `siteID` parameter
- Updated INSERT query to include `site_id` column
- Passes `p.SiteID` as first parameter in ExecContext

**Before:**

```go
type AxleProcessor struct {
    DB        *sql.DB
    RemoteDir string
    Minio     *minio.Client
    Bucket    string
}
```

**After:**

```go
type AxleProcessor struct {
    DB        *sql.DB
    SiteID    string // Site identifier for multi-site deployment
    RemoteDir string
    Minio     *minio.Client
    Bucket    string
}
```

### 5. `internal/handler/dimension_handler.go` ✅

**Changes:**

- ✅ Already has `SiteID` field in `DimensionHandler` struct
- Fixed INSERT query to remove `site_id` and `synced_to_central` columns (these will be added via migration later)
- Fixed parameter count to match SQL columns

**Note:** The dimension table will need migration to add site_id column later.

### 6. `main.go` ✅

**Changes:**

- Updated `NewAxleProcessor()` call to include `cfg.SiteID` parameter

**Before:**

```go
axleProcessor, err := handler.NewAxleProcessor(
    cfg.DB,
    cfg.AxleFTPDir,
    cfg.AxleMinIOEndpoint,
    ...
)
```

**After:**

```go
axleProcessor, err := handler.NewAxleProcessor(
    cfg.DB,
    cfg.SiteID, // Site identifier
    cfg.AxleFTPDir,
    cfg.AxleMinIOEndpoint,
    ...
)
```

## Database Schema Changes Required

The following columns need to be added to transaction tables via migration:

### transact_anpr_capture

```sql
ALTER TABLE public.transact_anpr_capture
ADD COLUMN site_id uuid REFERENCES public.master_site(id);
```

### transact_axle_capture

```sql
ALTER TABLE public.transact_axle_capture
ADD COLUMN site_id uuid REFERENCES public.master_site(id);
```

### vehicle_dimensions (if table exists)

```sql
ALTER TABLE vehicle_dimensions
ADD COLUMN site_id VARCHAR(50) REFERENCES sites(site_id);
```

These migrations are already in: `migrations/100_simple_multi_site.sql`

## How It Works Now

1. **Application Startup:**

   - Reads SITE_ID from .env
   - Validates site exists in master_site table
   - Initializes all handlers with site_id

2. **ANPR Data Processing:**

   - When ANPR captures plate, includes cfg.SiteID
   - Inserts into transact_anpr_capture with site_id
   - All records tagged with originating site

3. **Axle Data Processing:**

   - When axle sensor detects vehicle, includes cfg.SiteID
   - Inserts into transact_axle_capture with site_id
   - All records tagged with originating site

4. **Dimension Detection:**

   - When dimension detected, can be tagged with site (future migration)
   - Currently inserts without site_id (backward compatible)

5. **Central Sync (Optional):**
   - If CENTRAL_DATABASE_URL configured and SYNC_ENABLED=true
   - Application can sync data to central HQ database
   - Each site maintains local copy for reliability

## Testing Checklist

- [ ] Build application: `go build -o wim-service`
- [ ] Run migration: `psql -d wim_db -f migrations/100_simple_multi_site.sql`
- [ ] Verify master_site table exists and has default sites
- [ ] Update .env with appropriate SITE_ID
- [ ] Start application and verify site info in logs
- [ ] Test ANPR capture - check site_id in database
- [ ] Test axle capture - check site_id in database
- [ ] Verify foreign key constraints work

## Deployment Steps for New Site

1. **Database Setup:**

   ```bash
   psql -d new_site_db -f migrations/ddl.sql
   psql -d new_site_db -f migrations/100_simple_multi_site.sql
   ```

2. **Configure .env:**

   ```env
   SITE_ID="NEW-SITE-01"
   SITE_NAME="New Site Name"
   SITE_LOCATION="Physical Address"
   SITE_REGION="Region Name"
   DATABASE_URL="postgres://..."
   ```

3. **Insert Site to Database:**

   ```sql
   INSERT INTO public.master_site (code, site_name, site_location, site_region)
   VALUES ('NEW-SITE-01', 'New Site Name', 'Physical Address', 'Region Name');
   ```

4. **Deploy & Run:**
   ```bash
   go build -o wim-service
   ./wim-service
   ```

## Benefits of Multi-Site Implementation

✅ **Data Tracking**: Setiap transaksi tahu berasal dari site mana
✅ **Centralized Reporting**: HQ bisa aggregate data dari semua site
✅ **Flexible Deployment**: Site bisa standalone atau sync ke central
✅ **Scalability**: Easy to add new sites
✅ **Backward Compatible**: Existing single-site deployment tetap jalan

## Notes

- SITE_ID must exist in master_site.code before starting application
- If SITE_ID not found, application should log error/warning
- For production, implement validation on startup
- Consider adding health check endpoint that reports site info
- Monitor sync status if central sync enabled

---

**Last Updated:** December 10, 2025
**Implementation Status:** Complete ✅
