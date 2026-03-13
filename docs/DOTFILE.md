# Dotfile Format

CardBot writes a `.cardbot` file to the root of processed cards to track copy history.

## Location

`<mount-point>/.cardbot`

Example: `/Volumes/NIKON Z 9  /.cardbot`

## Schema (v2)

```json
{
  "$schema": "cardbot-dotfile-v2",
  "copies": [
    {
      "mode": "selects",
      "timestamp": "2026-03-12T12:31:08-07:00",
      "destination": "/Users/user/Pictures/CardBot",
      "files_copied": 42,
      "bytes_copied": 1234567890,
      "verified": true,
      "cardbot_version": "0.1.9"
    },
    {
      "mode": "all",
      "timestamp": "2026-03-12T14:22:10-07:00",
      "destination": "/Users/user/Pictures/CardBot",
      "files_copied": 3051,
      "bytes_copied": 96424837120,
      "verified": true,
      "cardbot_version": "0.1.9"
    }
  ]
}
```

## Fields

### Top-Level

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `$schema` | string | Yes | Schema version (`cardbot-dotfile-v2`) |
| `copies` | array | Yes | Array of copy entries, one per mode |

### Copy Entry

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `mode` | string | Yes | Copy mode: `all`, `selects`, `photos`, or `videos` |
| `timestamp` | string | Yes | ISO 8601 / RFC 3339 timestamp when copy completed |
| `destination` | string | Yes | Absolute path where files were copied |
| `files_copied` | number | Yes | Count of files copied |
| `bytes_copied` | number | Yes | Total bytes copied |
| `verified` | boolean | Yes | Whether size verification passed |
| `cardbot_version` | string | Yes | Version of CardBot that created the entry |

## Copy Modes

| Mode | Description | Status |
|------|-------------|--------|
| `all` | All files from card | ✅ Implemented |
| `selects` | Only star-rated images | ✅ Implemented |
| `videos` | Only video files | ✅ Implemented |
| `photos` | Only photo files | ✅ Implemented |

## Behavior

### Writing

- Dotfile is written **only after** successful copy and size verification
- If write fails (read-only card), copy still succeeds — card shows as "New" next insert
- Write is atomic: temp file (`.cardbot.tmp`) + rename

### Reading

- Presence of valid `.cardbot` → status "Copy completed on 2026-03-12T12:31:05"
- Absence or parse error → status "New"

### Display

```
  Status:   New
  Status:   Copy completed on 2026-03-11T21:31:05
```

## Future Extensibility

Schema version enables adding fields without breaking older CardBot versions:

- **v1**: Copy stats (legacy single-mode format)
- **v2** (current): Array format for tracking multi-mode partial copies
- **v3** (future): File manifest with hashes for incremental copy / file integrity
- **v4** (future): Card identification (serial, model), multiple destinations

Unknown schema versions: treat card as "New", warn user, don't modify file.

## Security Considerations

- Dotfile contains local filesystem paths (destination reveals username/directory structure)
- No sensitive data (passwords, credentials)
- Card may be shared: paths are visible to anyone reading the card
- Acceptable risk: paths alone don't compromise security
