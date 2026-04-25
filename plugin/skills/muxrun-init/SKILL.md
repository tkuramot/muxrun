---
name: muxrun-init
description: Analyze the current repository and generate a tailored muxrun.toml. Use when the user wants to start using muxrun in a project that has no config yet, or asks to "set up muxrun" / "generate muxrun.toml" / "initialize muxrun".
---

# muxrun-init

Generate a working `muxrun.toml` for the current repository by inspecting the codebase, then validate it with `muxrun check`.

## When to invoke

- The user asks to set up / initialize / bootstrap muxrun in a project.
- No `muxrun.toml` exists in the current directory or any ancestor.
- The user wants a starter config they can edit.

If a `muxrun.toml` already exists, ask whether to overwrite, augment, or abort before proceeding.

## Procedure

1. **Confirm prerequisites.**
   - Run `muxrun --version` to confirm the CLI is installed. If missing, point the user at the install instructions in `README.md` and stop.
   - Run `find . -maxdepth 3 -name muxrun.toml` to detect existing configs.

2. **Discover runnable processes.** Investigate the repository for long-running dev processes. Useful signals:
   - `package.json` → `scripts.dev`, `scripts.start`, `scripts.watch` (per-package in monorepos: check `pnpm-workspace.yaml`, `turbo.json`, `nx.json`, `lerna.json`, top-level `workspaces`).
   - `Procfile`, `Procfile.dev` → each line is an app.
   - `docker-compose.yml` / `compose.yml` → services with `command` and `working_dir` (note: prefer `/muxrun-import-compose` if Compose is the source of truth).
   - `Makefile` → targets like `dev`, `run`, `serve`, `watch`.
   - Go: `main.go` files (`find . -name main.go -not -path '*/vendor/*'`); each is a candidate app.
   - Python: `manage.py runserver`, `uvicorn`/`fastapi` entrypoints in `pyproject.toml`/`setup.cfg`.
   - Rust: `[[bin]]` entries in `Cargo.toml`.
   - Static frontends: `vite`, `next`, `astro`, `remix`, `nuxt` configs.
   - `.env`, `.env.example` to understand required vars (do NOT copy secrets into the config).

3. **Group apps sensibly.** A group is a tmux session and shares one `dir`. Reasonable defaults:
   - One group per top-level concern: `backend`, `frontend`, `infra`, `worker`.
   - In a monorepo, a group per workspace package is also valid; prefer the smaller number of groups that still keeps `dir` shared.
   - Do not put apps with different working directories in the same group — split them.

4. **Decide watch settings.** Enable `watch` only for apps that benefit from live restart (servers, workers). Leave `watch = false` (the default — omit the field) for one-shot or already-self-reloading processes (`vite`, `next dev`, `nodemon`, `air`).
   - When enabling watch, add an `exclude` list. Always start from project signals: read `.gitignore`, look for `node_modules/`, `dist/`, `build/`, `.next/`, `target/`, `vendor/`, `__pycache__/`, test fixtures (`testdata/`), generated mocks. Exclude patterns are Go regex matched against paths relative to the group `dir`.
   - Common defaults to consider: `_test\\.go$`, `mock_.*\\.go$`, `\\.tmp$`, `node_modules/`, `\\.next/`, `dist/`, `target/`.

5. **Write `muxrun.toml`.** Place it at the repository root (or wherever the user is). Use relative `dir = "."` style paths so the config travels well across worktrees (see `docs/config.md`). Naming rules: group/app names must match `^[a-zA-Z0-9_-]+$` with no duplicates within scope.

6. **Validate.** Run `muxrun check` (or `muxrun -c <path> check`). Fix any reported errors. If `muxrun check` is unavailable, validate by re-reading the file and confirming the rules in `docs/config.md`.

7. **Report.** Show the user:
   - The generated config and where it was written.
   - Apps you considered but skipped, with one-line reasons (e.g., "ran tests, not a long-running process").
   - Suggested next step: `muxrun up`.

## Reference

The full configuration schema, validation rules, and watch semantics are in [`config.md`](./config.md) (symlink to the canonical docs). Treat that file as the source of truth — do not invent fields.

## Anti-patterns

- Do not invent a `port`, `env`, or `depends_on` field. Those don't exist in `muxrun.toml`.
- Do not enable watch on processes that already have live reload — you'll fight the inner loop.
- Do not group apps with different `dir` values together.
- Do not include secrets from `.env` in the generated config.
