# psql-migrate
PostgreSQL database migrations. CLI and Go library.

PSQL-Migrate reads schema & migrations from filesystem and applies them in the correct order to a psql database.

This tool was created for my POC services. Do not use this for a serious application. For production use I would suggest [golang-migrate](https://github.com/golang-migrate/migrate) or a more serious migration tool [skeema](https://www.skeema.io/).

## CLI usage
- Handles ctrl+c (SIGINT) gracefully.

### Docker usage
Build the docker image with `make build-docker`
```bash
$ docker run -i -t -v {{ path/to/sql/directory }}:/sql --network host psql-migrate
    -s=/sql -d postgres://localhost:5432/database -v
```

## Getting Started
Before you start you should create the folder structure along with db.version file.

When psql-migrate is ran the folders will be executed in the following order `ddl` -> `dml` -> `migrations` with subfolders and subfiles executed in alphabetical order.

### Directory & file structure
Example of the directory & file structure with optional folders `ddl`, `dml` or `migrations`.
```
sql/
  ddl/
    table1.sql
    table2.sql
    ...
  dml/
    statement1.sql
    statement2.sql
    ...
  migrations/
    0.0.1/
      up/
        migration1.sql
        migration2.sql
        ...
    ...
  db.version
```

#### Optional directories
- `ddl` contains ddl statements (subdirectories are acceptable).
- `dml` contains dml statements (subdirectories are acceptable).
- `migrations` must contain subdirectories named with a valid semver `{major}.{minor}.{patch}` with an up subdirectory.

#### DB Version file
This file is required and it must only contain a valid semver value `{major}.{minor}.{patch}`

## Migrations
This service will only execute the up migrations, down is not supported.

It will execute the migrations greater than the last database db_version and the version equal to the db.version file.
