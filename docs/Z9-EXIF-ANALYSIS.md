# Nikon Z9 EXIF Analysis

**Date:** 2026-03-14  
**Camera:** Nikon Z9 (Body 1)  
**Firmware:** Unknown (production unit)

## Findings

### Subsecond Timestamp Precision

The Z9 writes **2-digit subsecond precision** in EXIF:

```
Date/Time Original    : 2025:10:17 14:27:45
Sub Sec Time Original : 50
```

This represents **0.50 seconds** (1/100 second resolution).

### Burst Timing Analysis

Z9 burst rate: Up to 20fps (1 frame every 0.05 seconds)

At 20fps with 0.01s EXIF precision:
- Shot 1 at t=0.000s → EXIF shows ".00"
- Shot 2 at t=0.050s → EXIF shows ".05"
- Shot 3 at t=0.100s → EXIF shows ".10"

**Critical finding:** With 0.01s precision and 0.05s burst interval, **multiple shots can share the same subsecond timestamp** if burst timing aligns with 1/100s windows.

Sample values from real files:
```
50, 26, 44, 61, 94, 23, 48, 47, 96, 26
```

### Conclusion for CardBot

**Subsecond timestamp alone is INSUFFICIENT for collision-free naming.**

Even with subsecond precision:
- Two cameras can fire in same 1/100s window
- Same camera burst can produce identical subsecond values
- Sequence number (0001-9999) is still required

**Recommendation:** Use camera prefix + sequence (`Z9_260314T143052_0001.NEF`) rather than subsecond timestamp.

## Full EXIF Sample

```
Make                            : NIKON CORPORATION
Camera Model Name               : NIKON Z 9
Date/Time Original              : 2025:10:17 14:27:45
Sub Sec Time Original           : 50
```

## Implications for 0.4.0 Multi-Camera

- Camera extraction: Use EXIF Model field, clean to "Z9"
- Dual-Z9 edge case: Body has same Make/Model - needs serial number or user suffix
- Subsecond approach: **Rejected** - insufficient precision
