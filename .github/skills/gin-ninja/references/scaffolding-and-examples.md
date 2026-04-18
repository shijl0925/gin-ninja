# Scaffolding and examples

## CLI entry points

The repository ships `gin-ninja-cli` in `cmd/gin-ninja-cli/`.

For focused `startproject` / `startapp` work, prefer the dedicated skill: [`../../gin-ninja-scaffold/SKILL.md`](../../gin-ninja-scaffold/SKILL.md).

Common commands:

- `gin-ninja-cli startproject <name> -module <go-module>`
- `gin-ninja-cli startapp <name>`
- `gin-ninja-cli init`
- `gin-ninja-cli generate crud -model <Name> -model-file <path>`
- `gin-ninja-cli makemigrations`
- `gin-ninja-cli migrate`
- `gin-ninja-cli showmigrations`
- `gin-ninja-cli sqlmigrate <migration>`

Use the CLI when the user asks for scaffolding, CRUD boilerplate, or migration workflows instead of hand-writing everything from scratch.

## Example selection guide

- `examples/basic/`: minimal typed CRUD API with SQLite, pagination, repo usage, and a health route
- `examples/users/`: focused auth + users application wiring
- `examples/features/`: targeted feature demo on top of the shared full app package
- `examples/admin/`: focused admin entry point
- `examples/full/`: the broadest end-to-end example with versioned docs, admin UI, config, and richer feature coverage

## Decision guide

- Need the smallest runnable shape -> start from `examples/basic/`
- Need project structure, config, logger, and bootstrap wiring -> inspect `examples/users/` or `examples/full/`
- Need versioning, admin, or richer framework features -> inspect `examples/full/`
- Need command-line project generation -> use `gin-ninja-cli startproject`
- Need a new domain package inside an existing project -> use `gin-ninja-cli startapp`
- Need CRUD boilerplate from a model -> use `gin-ninja-cli generate crud`

## Practical workflow

1. Decide whether scaffolding or targeted edits are faster.
2. If scaffolding, use the CLI first and then adjust the generated code.
3. If editing an existing service, copy patterns from the closest example instead of inventing new structure.
4. Keep docs endpoints and config wiring intact while extending the API.
