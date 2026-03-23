# create-config

Generate a valid `muxrun.toml` configuration file by interviewing the user about their project setup.

## Instructions

1. Read the configuration reference at `config.md` (in this skill's directory) to understand the full schema, field definitions, and validation rules.

2. Ask the user about their project:
   - How many groups of applications do they have? (e.g., backend, frontend, infrastructure)
   - For each group: what is the working directory?
   - For each group: what applications run in it? (name and command)
   - For each app: should file watching be enabled? If so, are there exclude patterns?

3. Generate a `muxrun.toml` file that:
   - Follows the exact TOML structure described in config.md
   - Passes all validation rules (required fields, naming conventions, valid regex patterns)
   - Uses inline table format for simple watch configs (`watch = { enabled = true }`)
   - Uses multi-line format for watch configs with multiple exclude patterns

4. Write the file to the project root (or the path the user specifies).

5. Suggest running `muxrun check` to validate the generated config.
