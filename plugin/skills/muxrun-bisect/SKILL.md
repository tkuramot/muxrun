---
name: muxrun-bisect
description: Run a git bisect against a behavior reproducible only with muxrun-managed dev servers (e.g., "the dev server started crashing on startup", "the API began returning 500s some time last week"). Use when the user wants to find the commit that introduced a runtime regression that requires a live process to observe.
---

# muxrun-bisect

Drive `git bisect` where the "good/bad" verdict requires bringing the dev stack up via muxrun and checking a runtime signal — not just running tests. Most regressions can be bisected with `git bisect run go test ...` or similar; use this skill only when a live process is required.

## When to invoke

- The user says "it used to work, now it crashes on startup" / "started returning 500s" / "the page is broken at runtime" and the regression is not caught by the test suite.
- A specific app under muxrun must be running to detect the problem.

If the regression IS caught by an existing test, recommend `git bisect run <test command>` directly and skip this skill.

## Pre-flight

1. **Confirm reproducibility.** Run `muxrun up`, observe the bug at HEAD. If you can't reproduce on the current branch, do not bisect — go gather more info.
2. **Identify a known-good ref.** Ask the user for a commit / tag / date when it definitely worked. Do `git log --oneline <good>..HEAD` to confirm there's a window to bisect.
3. **Check working tree is clean.** `git status`. If dirty, ask the user to stash or commit before proceeding — bisect will check out arbitrary commits.
4. **Decide the verdict signal.** Pick exactly one objective check, e.g.:
   - `muxrun ps` shows app X as running for >10s without exit.
   - `curl -fsS http://localhost:3000/health` returns 200.
   - A specific log line appears (or doesn't) in `muxrun logs <group> <app>`.
   - The daemon log shows no panics in the first 30s after startup.

   Write this down explicitly with the user before starting.

## Bisect loop

1. `muxrun down` (clean slate).
2. `git bisect start && git bisect bad <bad-ref> && git bisect good <good-ref>`.
3. For each commit git checks out:
   1. Run any required setup (`go build`, `npm install`, migrations) — only if a setup change at this commit makes it necessary; otherwise skip to keep cycles fast.
   2. `muxrun up`.
   3. Wait for the app under test to reach its steady state (use a fixed delay, e.g. `sleep 10`, or poll the verdict signal up to a timeout).
   4. Apply the verdict check.
   5. `muxrun down`.
   6. Run `git bisect good` or `git bisect bad` based on the result.
   7. If startup itself fails ambiguously (config errors, missing deps unrelated to the bug), `git bisect skip` and tell the user.
4. When git reports the first bad commit, report:
   - The commit SHA, author, date, and subject.
   - The diff (`git show <sha>`) — focused on files most likely related to the verdict signal.
   - A short hypothesis tying the diff to the symptom.
5. Always end with `git bisect reset` so the user lands back on their original branch.

## Automation option

If the verdict signal is a single command that returns 0/1, you can hand the loop to git directly:

```bash
git bisect start <bad> <good>
git bisect run sh -c '
  muxrun down >/dev/null 2>&1
  muxrun up >/dev/null 2>&1 || exit 125   # 125 = skip (cannot test)
  for i in $(seq 1 30); do
    curl -fsS http://localhost:3000/health && { muxrun down; exit 0; }
    sleep 1
  done
  muxrun down
  exit 1
'
```

Use exit code **125** for "this commit can't be tested" — git bisect skips it. **0** = good, **1** = bad. Show this script to the user before running it; they may want to tweak the timeout or signal.

## Action policy

- Ask before starting: bisect rewinds the working tree across many commits and runs `muxrun up`/`down` repeatedly. Confirm there's no uncommitted work and no other muxrun session running the same config in another worktree.
- Always `git bisect reset` at the end, even on failure.
- Never force-checkout over uncommitted changes.

## Anti-patterns

- Don't bisect when a faster signal (a test, a static check) exists.
- Don't use a flaky verdict signal — one false reading derails the whole bisect.
- Don't leave `muxrun up` running after a bisect step; always `down` between commits to avoid stale state polluting the next verdict.
