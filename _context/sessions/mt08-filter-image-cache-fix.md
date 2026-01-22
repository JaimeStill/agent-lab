# MT08: Filter Image Cache Fix

**Date**: 2026-01-22
**Type**: Bug Fix
**Branch**: `mt08-filter-image-cache-fix`

## Problem

The `classify-docs` workflow stopped detecting security markings. Documents that previously classified correctly were returning empty `markings_found` arrays and defaulting to `UNCLASSIFIED`.

## Root Cause

The image caching system in `internal/images/repository.go` used `WhereEquals` for nullable filter columns (`Quality`, `Brightness`, `Contrast`, `Saturation`, `Rotation`, `Background`) in the `findExisting` method.

`WhereEquals` ignores nil values entirely rather than adding a `column IS NULL` condition. This caused the cache lookup query to omit filter criteria when render options didn't specify filters:

```sql
-- Actual query (missing filter conditions)
SELECT ... FROM images
WHERE document_id = ? AND page_number = ? AND format = ? AND dpi = ?
LIMIT 1

-- Expected query
SELECT ... FROM images
WHERE document_id = ? AND page_number = ? AND format = ? AND dpi = ?
  AND brightness IS NULL AND contrast IS NULL AND saturation IS NULL ...
LIMIT 1
```

The query returned the first arbitrary image matching document/page/format/dpi, which happened to be a previously-rendered enhanced image with `brightness: 10` (nearly black). The vision model couldn't detect any markings on this darkened image.

## Fix

Changed `findExisting` to use `WhereNullable` for nullable columns, which properly adds `column IS NULL` conditions when values are nil:

```go
// Before
WhereEquals("Brightness", opts.Brightness)

// After
WhereNullable("Brightness", opts.Brightness)
```

## Files Changed

- `internal/images/repository.go:297-302` - Changed 6 `WhereEquals` calls to `WhereNullable`

## Verification

Ran the workflow against the same document that was failing. Results:
- Correct image returned (`57a82742...` with null filters)
- Markings detected: SECRET (header), SECRET (footer, faded), NOFORN (footer, faded)
- Classification: SECRET//NOFORN
- Confidence: 0.807
