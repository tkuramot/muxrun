---
name: debug-logs
description: Fetch and analyze logs from a running muxrun application
allowed-tools: Bash(muxrun *), Bash(tail *)
---

# debug-logs

Fetch logs from a running muxrun application and analyze them for errors or unexpected behavior.

## Instructions

1. Run `muxrun ps` to discover which groups and apps are currently running, and to identify the exact group and app names.

2. If the user has not specified which app to debug, ask them which app they want to investigate, referencing the names found in step 1.

3. Fetch logs with a line limit to avoid flooding the context:
   ```
   muxrun logs <group> <app> | tail -n 50
   ```
   Adjust the line count based on context — use fewer lines (20–30) for a quick check, more (100+) if the user reports an issue that may have occurred earlier.

4. If the logs alone are insufficient, read `muxrun.toml` to understand the app's configuration (command, working directory, watch settings) and use that context when analyzing.

5. Analyze the logs and report:
   - Any errors, panics, or stack traces
   - Unexpected exit codes or restart loops (relevant when `watch` is enabled)
   - Missing output that should be present

6. If the issue is not clear from a single fetch, iterate: fetch more lines, check a different app, or suggest the user reproduce the issue and fetch again.
