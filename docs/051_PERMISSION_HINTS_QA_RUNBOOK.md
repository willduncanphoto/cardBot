# 0.5.1 QA Runbook — Permission Hint Validation

Goal: validate daemon launch failure hints for:
- **051-003** Automation permission failures
- **051-004** Full Disk Access / permission denied failures

---

## Quick Start

```bash
make qa-051-permissions
```

This starts daemon mode, captures logs, and tails output live.

---

## Manual Steps

### A) Automation hint (051-003)

1. Open **System Settings → Privacy & Security → Automation**.
2. Revoke automation permission for your terminal app (or reset TCC and deny on prompt).
3. With daemon running, insert card.
4. Verify log includes:
   - `Launch failed: ...`
   - `Hint: ... Automation ...`

### B) Full Disk Access hint (051-004)

1. Create a scenario where daemon launch path returns permission denied / operation not permitted.
2. Trigger card insertion.
3. Verify log includes:
   - `Launch failed: ...`
   - `Hint: ... Full Disk Access ...`

---

## Pass/Fail

### PASS
- Automation-denied errors map to Automation hint.
- Permission-denied/EPERM errors map to Full Disk Access hint.

### FAIL
- No hint shown on known error patterns.
- Wrong hint shown for known error patterns.

---

## Output Artifacts

The helper script writes logs to:

- `/tmp/cardbot-qa-051-permissions-<timestamp>/daemon.log`

Capture relevant snippets in the QA run notes.
