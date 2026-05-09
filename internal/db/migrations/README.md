# SQL migrations (`golang-migrate`)

Schema changes live here as numbered pairs: `<version>_name.up.sql` and `<version>_name.down.sql`. Versions are sequential integers with the same digit width as existing files (e.g. `000002_after_000001`). The app applies these files automatically at startup via embedded assets in [`migrate.go`](../migrate.go)—this folder stays the single source of truth for DDL.

## Install the `migrate` CLI

Using Homebrew (macOS):

```bash
brew install golang-migrate
```

Or with Go:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Ensure `migrate` is on your `PATH` (for `go install`, your `GOPATH/bin` or `~/go/bin`).

## Create a new migration (recommended)

From the **repository root**, run `migrate create` with `-seq` so new files get the next integer prefix and match this project’s style:

```bash
migrate create -ext sql -dir internal/db/migrations -seq describe_your_change
```

This creates two empty files, for example:

- `000002_describe_your_change.up.sql`
- `000002_describe_your_change.down.sql`

Edit the `.up.sql` with your `CREATE` / `ALTER` / index changes, and the `.down.sql` with the reverse (`DROP`, etc.).

If you prefer to name files by hand, copy the pattern from `000001_*` and use the next version number—**do not** edit already-applied migrations on environments that have run them.

## Run migrations with the CLI (optional)

The service runs `Up()` on boot, but you can apply or inspect the same folder against Postgres using your DSN (same idea as `DATABASE_URL` in your config / `.env`):

```bash
export DATABASE_URL='postgres://user:pass@localhost:5432/dbname?sslmode=disable'

migrate -path internal/db/migrations -database "$DATABASE_URL" up
```

Other useful commands:

```bash
# Current version
migrate -path internal/db/migrations -database "$DATABASE_URL" version

# Roll back one step (uses the .down.sql for the current version)
migrate -path internal/db/migrations -database "$DATABASE_URL" down 1

# Force version if you need to fix schema_migrations after manual repair (use with care)
# migrate -path internal/db/migrations -database "$DATABASE_URL" force <version>
```

## After adding files

1. Run `go build ./...` so embed picks up new `*.sql` files under this directory.
2. Update [`models.go`](../models.go) (and any queries) so they match the new schema—migrations own the DDL.

## References

- [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)
- [Postgres URLs / query params](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING) (`sslmode`, etc.)
