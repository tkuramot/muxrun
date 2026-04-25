---
name: muxrun-tail
description: Stream a muxrun-managed app's logs and surface anomalies (panics, stack traces, 5xx responses, slow requests) in real time. Use when the user wants to "watch", "tail", "follow", or "monitor" an app's output, or asks to be alerted when something goes wrong.
---

# muxrun-tail

Run `muxrun logs --follow` against a chosen app, watch the stream, and call out interesting events. Unlike `/muxrun-triage-failure` (which acts on already-broken state), this skill is for live observation.

## When to invoke

- "Tail / watch / follow the logs of X."
- "Let me know if anything goes wrong while I run Y."
- The user is reproducing an intermittent issue and wants commentary, not a wall of output.

## Procedure

1. **Pick the target.**
   - If the user names `<group> <app>`, use it.
   - If they name only an app and it is unique across groups, infer.
   - Otherwise run `muxrun ps` and ask the user to choose. Don't guess silently.

2. **Confirm the app is running.** If `muxrun ps` shows it stopped, switch to `/muxrun-triage-failure` instead — there's nothing to tail.

3. **Start the follow.** Run `muxrun logs --follow <group> <app>` in the background so you can keep interacting with the user. Capture output in chunks rather than dumping everything.

4. **Anomaly patterns to surface.** Treat these as noteworthy and call them out with the originating log line:
   - Stack traces (Go `goroutine ... [running]`, Python `Traceback`, Node `at <function> (file:line)`).
   - Panics / fatal errors (`panic:`, `FATAL`, `unhandled exception`, `Segmentation fault`).
   - HTTP 5xx in access-log-shaped lines.
   - Slow-request markers (configurable thresholds; default >1s if a duration is logged).
   - Repeated identical errors (note the count instead of repeating each line).
   - Sudden silence after a high rate of output (possible deadlock).
   - Restart events from the watch daemon (process re-exec).

5. **Summarize, don't dump.** Default to:
   - One short status line per ~10 seconds of quiet output ("nominal, N requests, no errors").
   - Full quote of any anomaly line, with surrounding 2–3 lines of context.
   - A running tally of error categories.

6. **Stop conditions.** Stop following when:
   - The user says stop / cancel / done.
   - The app exits (process gone) — switch to triage and report.
   - You've collected what the user asked to see.

   Always tear down the background `muxrun logs --follow` cleanly when stopping.

## Multi-app

If the user wants to follow multiple apps at once, start one background follow per app and prefix each surfaced event with `[group/app]`. Keep the number small (≤3) to avoid noise.

## Anti-patterns

- Don't paste raw log streams of more than ~50 lines into the conversation — summarize.
- Don't silently drop the background process; always stop it when done.
- Don't add filters that hide warnings the user did not ask to hide.
