---
name: muxrun-triage-failure
description: Identify which muxrun-managed app(s) are failing or restart-looping, fetch their recent output, and pinpoint a likely cause. Use when the user says something is "broken", "not working", "crashing", "won't start", or asks "why is X failing?" about a running muxrun setup.
---

# muxrun-triage-failure

Find the failing app(s) automatically — do not make the user name them — then diagnose from logs.

## When to invoke

- The user reports a failure with vague scope: "the backend is broken", "something crashed", "muxrun keeps restarting things", "why isn't it working".
- After `muxrun up`, when the user expects things to be running but they aren't.
- During development when an app silently exits.

## Procedure

1. **Get the current state.** Run `muxrun ps`. The output is a table; columns include the group, app, status, PID, and elapsed time. Treat as failures any row that is:
   - Not running (process gone),
   - Recently exited (small "elapsed since exit" → restart loop),
   - Showing repeated short uptimes across consecutive `ps` calls.

   If `ps` shows nothing, run `muxrun check` first — the config may not resolve.

2. **Narrow the scope.**
   - If the user named a group/app, focus there.
   - Otherwise enumerate all failing rows and proceed in parallel.

3. **Pull logs for each failing app.** Use `muxrun logs <group> <app>` (without `--follow`) to grab the current pane buffer. This is the captured tmux pane output — it contains everything the app has printed since launch (or since the last restart).
   - For watch-enabled apps, also read the daemon log: `$TMPDIR/muxrun/daemon-<group>.log` — it records the restart cycle and fsnotify events.

4. **Analyze.** Look for:
   - Stack traces / panics / unhandled exceptions (last one is usually the cause).
   - "address already in use", "EADDRINUSE", "permission denied", "command not found".
   - Missing env vars (`undefined: SOMETHING`, `KeyError: 'FOO'`, `parse error`).
   - Module/dep errors (`Cannot find module`, `ImportError`, `package X is not in GOROOT`).
   - Watch-induced restart loops: a write event in the log immediately followed by a restart, repeated tightly. This points to a missing `exclude` pattern (see `/muxrun-perf` for tuning).
   - Exit on stdin EOF, or commands that expect a TTY but lose one in tmux.

5. **Cross-check the command.** Read the relevant `[[group.app]]` entry. Confirm `cmd` runs from `dir`, that it doesn't depend on shell aliases, and that any tools it invokes are on `PATH` for non-interactive shells.

6. **Report.**
   - One bullet per failing app: status, the root error line (quoted, with file:line if present), and the most likely cause.
   - Concrete fix suggestion. If the fix involves editing `muxrun.toml` (e.g., add `exclude` for a hot-reload loop), show the diff.
   - If the cause is in app code, point to the file and line — do not edit app source unless asked.

## Edge cases

- **Nothing is failing.** `muxrun ps` shows everything healthy but the user disagrees. Ask what they expected; check the actual app behavior (HTTP probe, log content) rather than just process state.
- **App is "running" but unresponsive.** Process alive but no recent log output and request hangs. Check the log buffer for the last meaningful line; suggest a manual `kill` + restart and watch for the same hang.
- **No logs at all.** App may exit before printing. Wrap the command temporarily with `sh -c '... 2>&1; echo EXIT=$?; sleep 1'` to capture the exit code, or run `cmd` directly in a terminal to see startup output.

## Anti-patterns

- Don't ask the user "which app?" without first running `muxrun ps`.
- Don't recommend `muxrun down && muxrun up` as the diagnosis — that hides the cause.
- Don't edit app source code as part of triage; report the cause and let the user decide.
