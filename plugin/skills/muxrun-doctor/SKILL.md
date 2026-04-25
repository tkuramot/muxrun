---
name: muxrun-doctor
description: Diagnose muxrun environment and runtime issues — missing tmux, stale daemon PIDs, leftover sessions, port conflicts, broken config. Use when "muxrun isn't working", commands hang or error out, or after an unclean shutdown / crash. Read-only by default; only takes destructive action with explicit user confirmation.
---

# muxrun-doctor

Run a top-to-bottom health check of the local muxrun setup and report findings. Suggest fixes; do not execute destructive cleanup without confirmation.

## When to invoke

- The user reports muxrun "isn't working", commands hang, `up` fails immediately, `ps` shows stale state, or daemons keep restarting.
- After a crash, system reboot, or `kill -9` of muxrun/tmux.
- Before filing a bug — to gather diagnostic info.

## Checks (run in this order)

### 1. CLI presence and version
- `command -v muxrun` and `muxrun --version`. If missing, refer to `README.md` install instructions.

### 2. tmux availability
- `command -v tmux` and `tmux -V`. muxrun requires tmux. If absent, instruct install (`brew install tmux` / package manager).
- `tmux ls 2>&1` — list all sessions. Note any `muxrun-*` sessions.

### 3. Config resolution
- `muxrun check` from the user's current dir. If it fails, surface the specific error.
- Report which config file was resolved (CWD walk-up, see `docs/config.md`).

### 4. Daemon state
muxrun stores daemon artifacts under `$TMPDIR/muxrun/` (macOS) or `/tmp/muxrun/` (Linux fallback). Resolve `$TMPDIR` first.
- `ls -la "$TMPDIR/muxrun/" 2>/dev/null` — list `daemon-<group>.pid` and `daemon-<group>.log` files.
- For each PID file: read it, then `ps -p <pid> -o pid,stat,etime,command` to check if the process is alive.
- **Stale PID file**: PID file exists but the process is gone. Note as a stale entry; suggest removing the PID file (do NOT remove without confirmation).
- **Orphan daemon**: process is alive but no corresponding tmux session (`muxrun-<group>`) exists. Suggest `kill <pid>` after confirming.
- **Mismatched daemon**: process command line doesn't look like `muxrun _daemon ...`. Flag and ask the user.

### 5. tmux session vs config drift
- For each group in the resolved config, check `tmux has-session -t muxrun-<group> 2>/dev/null`.
- For each existing `muxrun-*` session not in the current config, flag as "leftover from a different config / worktree".

### 6. Daemon log scan
- For each `daemon-<group>.log` in `$TMPDIR/muxrun/`, look at the tail (`tail -n 100`) for repeated restart loops, fsnotify errors, "too many open files", or panics.
- Report a one-line summary per daemon.

### 7. Port / resource hints (best-effort)
- If the user's config commands include obvious ports (e.g., `:3000`, `--port 8080`, `PORT=`), check `lsof -nP -iTCP:<port> -sTCP:LISTEN 2>/dev/null` for conflicts.
- Note: this is heuristic — only flag what is clearly relevant to the running config.

### 8. File descriptor / watch limits (when watch is enabled)
- macOS: `launchctl limit maxfiles` and `ulimit -n`.
- Linux: `cat /proc/sys/fs/inotify/max_user_watches`. fsnotify exhausts watches on large trees — recommend raising the limit or expanding `exclude` patterns when the count is low (< 524288 on Linux).

## Output

Produce a structured report:

```
## muxrun doctor

[OK] muxrun v0.x.x
[OK] tmux 3.x
[WARN] Stale daemon PID: $TMPDIR/muxrun/daemon-backend.pid -> 12345 (no such process)
[FAIL] muxrun check: <error>
...

### Suggested actions
1. Remove stale PID: rm "$TMPDIR/muxrun/daemon-backend.pid"   (needs confirmation)
2. ...
```

## Action policy

- **Read-only by default.** Doctor diagnoses; it does not clean up.
- **Confirm before destructive ops.** Removing PID files, killing daemons, killing tmux sessions, raising kernel limits — all require explicit user "yes". Show the exact command before running it.
- Never `kill -9` muxrun / tmux without first trying graceful (`muxrun down` → `kill <pid>` → only then escalate).

## Anti-patterns

- Don't invent fixes that aren't supported by what you observed.
- Don't suggest `pkill -f muxrun` as a default — it can take down legitimate sessions in other worktrees.
- Don't remove `$TMPDIR/muxrun/` wholesale.
