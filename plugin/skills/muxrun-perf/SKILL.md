---
name: muxrun-perf
description: Diagnose and fix slow startup, restart loops, or excessive file-watch churn in a muxrun setup. Use when the user complains that muxrun is "slow", apps "keep restarting", "rebuilds too often", or that watch is "too aggressive / fires on everything".
---

# muxrun-perf

Profile how muxrun is behaving for the current config and propose concrete tuning — usually `exclude` patterns, sometimes splitting a group, sometimes turning watch off.

## When to invoke

- The user reports slowness, restart storms, or watch firing on irrelevant files.
- Build/rebuild cycles feel longer than expected after enabling `watch`.
- CPU is high while idle.

## Procedure

1. **Read the config.** Identify which apps have `watch.enabled = true` and their `exclude` lists. Note `dir` for each group — that's the watch root.

2. **Check restart frequency from daemon logs.**
   - For each `watch`-enabled group, read `$TMPDIR/muxrun/daemon-<group>.log`.
   - Count restarts in the last N minutes (`grep -c restart` or similar — match the actual log format used by the daemon).
   - Identify the file events that triggered each restart. A healthy loop is: edit → 1 event → 1 restart. A bad loop is: 1 edit → many events, or restarts triggered by files the user didn't edit (logs, build artifacts, generated code, editor swap files).

3. **Inventory the watch tree.**
   - From each group's `dir`, run `find . -type f | wc -l` to size the tree.
   - Estimate the share covered by current `exclude` patterns. If the tree is >10k files and `exclude` is empty/short, watch is almost certainly over-broad.
   - On Linux, compare to `cat /proc/sys/fs/inotify/max_user_watches` — exhaustion silently drops events.

4. **Detect likely offenders.** Look for these and suggest excluding them if not already:
   - VCS noise: `\\.git/`
   - Dependency stores: `node_modules/`, `vendor/`, `target/`, `.venv/`, `__pycache__/`
   - Build outputs: `dist/`, `build/`, `out/`, `\\.next/`, `\\.turbo/`, `\\.cache/`
   - Coverage / test artifacts: `coverage/`, `\\.nyc_output/`, `testdata/`
   - Logs and tmp: `\\.log$`, `\\.tmp$`, `\\.swp$`, `~$`
   - Generated code: project-specific (mocks, protobuf, codegen output) — read `.gitignore` for hints.
   - The app's own output written into `dir` (a classic self-restart loop — recommend writing artifacts to `/tmp` or excluding the path).

5. **Detect "wrong watch" cases.** Sometimes turning watch off is the right answer:
   - Commands that already self-reload (`vite`, `next dev`, `nodemon`, `air`, `cargo watch`, `mix phx.server`). The muxrun daemon stacking on top causes double restarts.
   - One-shot commands that should not be restarted at all.

6. **Detect group-level issues.**
   - A single group with many heavy apps where one slow restart starves the others — suggest splitting groups so they restart independently.
   - `dir` set to a parent that contains an entire monorepo when only a sub-tree is relevant — suggest narrowing `dir` per group.

7. **Propose a diff.** Show a concrete `muxrun.toml` patch with new `exclude` entries (Go regex, paths relative to the group `dir`), or a watch toggle, or a group split. Validate with `muxrun check`.

8. **Verify.**
   - After the user applies the change, do `muxrun down && muxrun up`, then re-read daemon logs after a short idle period and after one intentional edit. The expected outcome: zero restarts when idle, exactly one restart per intended edit.

## Reference

- Watch semantics and exclude regex syntax: [`config.md`](../muxrun-init/config.md) (or `docs/config.md`).

## Anti-patterns

- Don't propose `exclude = [".*"]`; that's the same as disabling watch — just set `watch = false`.
- Don't add patterns that match source files the user actively edits.
- Don't tune blindly without reading the daemon log — the actual events are the ground truth.
- Don't disable watch on every app to "make it faster" — first identify which ones are actually noisy.
