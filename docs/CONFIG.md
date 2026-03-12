# Configuration File

Configuration file location: `~/.config/cardbot/config.json`

Permissions are set to `0600` on creation (owner read/write only).

## Schema

```json
{
  "$schema": "cardbot-config-v1",
  "destination": {
    "path": "~/Pictures/CardBot"
  },
  "output": {
    "color": true,
    "quiet": false
  },
  "advanced": {
    "buffer_size_kb": 256,
    "exif_workers": 4,
    "log_file": ""
  }
}
```

## Fields

### destination

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `path` | string | `"~/Pictures/CardBot"` | Base folder for copied cards |

Path expansion:
- `~` expanded to `$HOME`
- Relative paths resolved from working directory

### output

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `color` | boolean | `true` | Enable ANSI color codes |
| `quiet` | boolean | `false` | Suppress non-error output |

### advanced

| Field | Type | Default | Range | Description |
|-------|------|---------|-------|-------------|
| `buffer_size_kb` | number | `256` | 64-4096 | Copy buffer size in KB |
| `exif_workers` | number | `4` | 1-16 | Concurrent EXIF parsing workers |
| `log_file` | string | `"~/.cardbot/cardbot.log"` | path or "" | Log file path (empty = no logging) |

## Validation Rules

On load, invalid values are replaced with defaults and a warning is printed:

| Field | Invalid If | Action |
|-------|------------|--------|
| `buffer_size_kb` | < 64 or > 4096 | Clamp to range |
| `exif_workers` | < 1 or > 16 | Clamp to range |

## Version Field

Config file includes implicit version via schema field. Future versions will increment schema and provide migration.

```json
{
  "$schema": "cardbot-config-v2"
}
```

Migration behavior:
- Unknown schema version: ignore file, use defaults, warn user
- Older schema: auto-migrate to current, backup old version

## Example Configurations

### Minimal

```json
{
  "output": {
    "color": false
  }
}
```

Unspecified fields use defaults.

### Studio Workflow

```json
{
  "output": {
    "color": false
  },
  "advanced": {
    "exif_workers": 8,
    "log_file": "~/.cardbot/cardbot.log"
  }
}
```

## Configuration Priority

1. CLI flags (highest priority)
2. Config file (`~/.config/cardbot/config.json`)
3. Built-in defaults
