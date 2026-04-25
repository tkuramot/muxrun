---
name: muxrun-import-compose
description: Convert an existing docker-compose.yml, Procfile, or foreman/honcho config into a muxrun.toml. Use when the user wants to migrate from Compose/Procfile to muxrun, or asks to "import compose" / "convert Procfile" / "use muxrun instead of compose".
---

# muxrun-import-compose

Translate an existing process orchestrator file into `muxrun.toml`. muxrun runs commands directly on the host through tmux — it does not run containers — so the conversion is lossy for anything container-specific. Be explicit about what cannot be carried over.

## When to invoke

- The user explicitly asks to migrate from `docker-compose.yml`, `compose.yml`, `Procfile`, `Procfile.dev`, or a foreman/honcho/overmind config to muxrun.
- The user mentions wanting a non-container local dev workflow but already has a Compose file.

If both Compose and a Procfile exist, ask which one is canonical.

## Procedure

1. **Locate source files.** Check for `docker-compose.yml`, `compose.yml`, `compose.yaml`, `docker-compose.*.yml`, `Procfile`, `Procfile.dev`. If multiple Compose files exist (e.g., `docker-compose.override.yml`), confirm which to use, or merge after asking.

2. **Translation rules.**

   ### Procfile → muxrun.toml
   - Each line `name: command` becomes one `[[group.app]]` with `name = "<name>"`, `cmd = "<command>"`.
   - Default to a single group named after the project, with `dir = "."`.
   - `Procfile.dev` typically targets local dev — enable `watch` on apps whose command suggests a server/worker without built-in reload.

   ### docker-compose → muxrun.toml
   For each service:
   - **Group placement.** Default to one group per `working_dir` family. If all services share a working dir, one group is fine.
   - **`name`.** Use the service key. If it contains characters outside `^[a-zA-Z0-9_-]+$`, sanitize and tell the user.
   - **`cmd`.** Use `command:` if present. If only `image:` is set, ask the user to supply a host command — there is no host-side command to run. Do not fabricate one.
   - **`dir` (group).** Use `working_dir` if uniform across the group, else `.`.
   - **`environment` / `env_file`.** muxrun has no env field. Tell the user to either source vars in their shell or wrap the command (e.g., `cmd = "env $(cat .env | xargs) <real cmd>"`). Do NOT inline secrets.
   - **`ports`.** Informational only — not represented in muxrun. Note them in the report.
   - **`depends_on`.** Not represented. muxrun starts apps in declaration order without dependency awareness; advise the user to order apps so dependencies start first, or split into groups they bring up sequentially.
   - **`volumes`.** Irrelevant on host (the host filesystem is the source). Drop them.
   - **`build:` / Dockerfile.** Cannot be expressed. If the service truly needs a container, recommend keeping it on Compose and running only host services through muxrun.

3. **Watch defaults.** Enable `watch = { enabled = true, exclude = [...] }` only when the original command implies a server/worker without its own reloader. Use `.gitignore` plus the standard ignore set (`node_modules/`, `dist/`, `target/`, `__pycache__/`, generated dirs).

4. **Write the file.** Use relative `dir`. Validate with `muxrun check` and fix any errors.

5. **Report.** Always include:
   - A side-by-side mapping: source service/process → muxrun group/app.
   - A "**not migrated**" list: build steps, volumes, ports, depends_on, env vars, healthchecks. Be specific about each one and what the user should do instead.
   - Whether running through muxrun is appropriate at all (if everything is image-based and there is no host-runnable command, recommend staying on Compose).

## Reference

Schema and validation rules: [`config.md`](./config.md).

## Anti-patterns

- Do not invent `cmd` for services that only declare `image:`.
- Do not collapse services with different `working_dir` into one group.
- Do not silently drop env vars / ports / depends_on. List them as not-migrated so the user can decide.
- Do not inline secrets from `env_file` into the generated config.
