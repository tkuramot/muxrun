# create-config

Generate a valid `muxrun.toml` configuration file by analyzing the project's directory structure and inferring how to run each service.

## Instructions

1. Read the configuration reference at `config.md` (in this skill's directory) to understand the full schema, field definitions, and validation rules.

2. Explore the project's directory structure to auto-generate a `muxrun.toml`:
   - Identify runnable services by looking for `package.json`, `go.mod`, `Cargo.toml`, `Makefile`, `docker-compose.yml`, and similar project markers.
   - Infer start commands from scripts and project files.
   - Determine the grouping strategy from the directory structure. A group is a unit that shares a working directory (`dir`), and may contain multiple apps.
   - Use relative paths for `dir` (e.g., `"."` or `"./backend"`) rather than absolute paths. Relative paths are resolved from the `muxrun.toml` location, so the config works correctly when copied into git worktrees.
   - For file watching: tools that provide their own hot-reload typically don't need muxrun's `watch` enabled. Enable `watch` for services that lack built-in reload.
   - Follow the exact TOML structure described in config.md. Use inline table format for simple watch configs (`watch = { enabled = true }`) and multi-line format when there are multiple exclude patterns.

3. Present the generated config to the user and ask for confirmation before writing.

4. If the user requests changes, adjust the config through conversation until they are satisfied.

5. Once confirmed, write `muxrun.toml` to the project root (or the path the user specifies) and suggest running `muxrun check` to validate it.
