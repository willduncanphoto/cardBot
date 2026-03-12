# Dotfile Format

CardBot writes a `.cardbot` file to the root of processed cards to track copy history.

## Location

`<mount-point>/.cardbot`

Example: `/Volumes/NIKON Z9/.cardbot`

## Schema

```json
{
  "$schema": "cardbot-dotfile-v1",
  "last_copied": "2025-03-08T21:22:18-07:00",
  "mode": "all",
  "destination": "/Users/user/Pictures/Ingest/CardBot_250308_001",
  "card": {
    "serial": "0xA1B2C3D4",
    "model": "SanDisk Extreme Pro"
  },
  "files": [
    {
      "path": "DCIM/100NIKON/DSC_0001.NEF",
      "size": 52428800,
      "hash": "a1b2c3d4e5f67890"
    }
  ],
  "stats": {
    "files_copied": 1247,
    "bytes_copied": 152882347520,
    "verified": true
  },
  "cardbot_version": "0.3.0"
}
```

## Fields

### Required

| Field | Type | Description |
|-------|------|-------------|
| `$schema` | string | Schema version for forward compatibility |
| `last_copied` | string | ISO 8601 timestamp with timezone when copy completed |
| `mode` | string | Copy mode used: `all`, `selects`, `videos`, `photos` |
| `destination` | string | Absolute path where files were copied |
| `files` | array | List of copied files with size and hash |

### Optional

| Field | Type | Description |
|-------|------|-------------|
| `card` | object | Card identification |
| `card.serial` | string | Card serial number (if available) |
| `card.model` | string | Card model name (if available) |
| `stats` | object | Copy statistics |
| `stats.files_copied` | number | Count of files copied |
| `stats.bytes_copied` | number | Total bytes copied |
| `stats.verified` | boolean | Whether size verification passed |
| `cardbot_version` | string | Version of CardBot that created the dotfile |

### File Entry

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Relative path from card root |
| `size` | number | File size in bytes |
| `hash` | string | xxhash64 checksum (16 hex chars) |

## Copy Modes

| Mode | Description |
|------|-------------|
| `all` | All files from card |
| `selects` | Only star-rated images |
| `videos` | Only video files |
| `photos` | Only photo files |

## Behavior

### Writing

- Dotfile is written **only after** successful copy and verification
- If write fails (read-only card), copy succeeds but card shows as "New" next insert
- Write is atomic: temp file + rename

### Reading

- Presence of `.cardbot` = card status "Processed"
- Absence = card status "New"
- Parse errors = treat as "New" with warning

### Display

```
Status:   Processed (copied 2 days ago)
```

Relative time calculated from `last_copied` to now.

## Future Extensibility

Schema version enables adding fields without breaking older CardBot versions:

- v1 (current): Basic tracking + file list with hashes (incremental-ready)
- v2 (future): Multiple destinations (backup copies)

Unknown schema versions: treat card as "New", warn user, don't modify file.

## Why File List + Hashes?

v1 includes full file manifest to enable future incremental copy:

- Only copy new/changed files on re-insert
- Detect card modifications (files added/removed)
- Verify destination integrity

## Example Dotfile

```json
{
  "$schema": "cardbot-dotfile-v1",
  "last_copied": "2025-03-08T21:22:18-07:00",
  "mode": "all",
  "destination": "/Users/user/Pictures/Ingest/CardBot_250308_001",
  "card": {
    "serial": "0xA1B2C3D4",
    "model": "SanDisk Extreme Pro"
  },
  "files": [
    {"path": "DCIM/100NIKON/DSC_0001.NEF", "size": 52428800, "hash": "a1b2c3d4e5f67890"},
    {"path": "DCIM/100NIKON/DSC_0002.NEF", "size": 52848216, "hash": "b2c3d4e5f6789012"},
    {"path": "DCIM/100NIKON/DSC_0003.JPG", "size": 8388608, "hash": "c3d4e5f678901234"}
  ],
  "stats": {
    "files_copied": 3,
    "bytes_copied": 113665624,
    "verified": true
  },
  "cardbot_version": "0.3.0"
}
```

## Security Considerations

- Dotfile contains local filesystem paths (may reveal username/structure)
- No sensitive data (passwords, credentials)
- Card may be shared: paths reveal destination to anyone reading card
- Acceptable risk: paths alone don't compromise security
