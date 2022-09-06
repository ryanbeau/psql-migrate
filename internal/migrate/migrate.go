package migrate

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/ryanbeau/psql-migrate/pkg/color"
	"github.com/ryanbeau/psql-migrate/pkg/db"
)

// Run the database migration.
func Run(ctx context.Context, conn db.Conn, root string) error {
	dbVersion, err := readDBVersion(ctx, conn)
	if err != nil {
		return fmt.Errorf("ERROR: Reading db version. %w", err)
	}

	file, err := ioutil.ReadFile(filepath.Join(root, "db.version"))
	if err != nil {
		return fmt.Errorf("ERROR: Reading schema version file. %w", err)
	}
	schemaVersion, err := db.Parse(string(file))
	if err != nil {
		return fmt.Errorf("ERROR: Reading schema version. %w", err)
	}

	color.Printlnf(color.Blue, "Database: v%s => Schema: v%s", dbVersion, schemaVersion)
	if schemaVersion.LessThanOrEq(dbVersion) {
		color.Println(color.Yellow, "No migration operation will be performed. Check settings.")
		return nil
	}

	files, err := getSchemaFiles(root, dbVersion, schemaVersion)
	if err != nil {
		return fmt.Errorf("ERROR: Getting sql files from path. %w", err)
	}

	err = executeMigrationsTx(ctx, conn, files)
	if err != nil {
		return fmt.Errorf("ERROR: Executing migration. %w", err)
	}

	err = updateDbVersion(ctx, conn, schemaVersion)
	if err != nil {
		return fmt.Errorf("ERROR: On update db_version. %w", err)
	}

	color.Printlnf(color.Blue, "Updated db_version to %s", schemaVersion)
	color.Printlnf(color.Green, "Migration finished successfully!")

	return nil
}

func readDBVersion(ctx context.Context, conn db.Conn) (db.Version, error) {
	var version db.Version

	//check if function exists or return zero value
	_, err := conn.Exec(ctx, "SELECT pg_get_functiondef('get_db_version()'::regprocedure);")
	if err != nil {
		var pge *pgconn.PgError
		if errors.As(err, &pge) && pge.SQLState() == pgerrcode.UndefinedFunction {
			return version, nil
		}
		return version, err
	}

	//get version or zero
	err = conn.QueryRow(context.Background(), "SELECT id, major, minor, patch, started_at, finished_at FROM get_db_version();").
		Scan(&version.ID, &version.Major, &version.Minor, &version.Patch, &version.StartedAt, &version.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return version, nil
	}
	return version, err
}

func updateDbVersion(ctx context.Context, conn db.Conn, version db.Version) error {
	//ensure function exists
	_, err := conn.Exec(ctx, "SELECT pg_get_functiondef('db.update_db_version(INT,INT,INT,TIMESTAMP,TIMESTAMP)'::regprocedure);")
	if err != nil {
		return fmt.Errorf("Cannot update on db_version: %v", err)
	}

	//start update version
	sql := "SELECT db.update_db_version($1,$2,$3);"
	_, err = conn.Exec(context.Background(), sql, version.Major, version.Minor, version.Patch)
	if err != nil {
		return err
	}
	return nil
}

func executeMigrationsTx(ctx context.Context, conn db.Conn, files []string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if !tx.Conn().IsClosed() {
			err = tx.Rollback(context.Background())
			if err == nil {
				color.Println(color.Blue, "INFO: Successfully performed rollback on transaction")
			} else if !errors.Is(err, pgx.ErrTxClosed) {
				color.Println(color.Yellow, err.Error())
				color.Println(color.Red, "ERROR: Unexpected error rolling back transaction")
			}
		}
	}()

	if err := executeMigrations(ctx, tx, files); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func executeMigrations(ctx context.Context, conn db.Conn, files []string) error {
	var lastDir string
	for _, file := range files {
		dir, _ := filepath.Split(file)
		if dir != lastDir {
			color.Printlnf(color.Cyan, "Running folder: %s", dir)
			lastDir = dir
		}
		color.Printlnf(color.White, "Executing file: %s", file)

		//read sql
		sql, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		//execute sql
		_, err = conn.Exec(ctx, string(sql))
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Detail != "" {
				color.Printlnf(color.Yellow, "DETAILS: %v", pgErr.Detail)
			}
			return err
		}
	}
	return nil
}

func getFiles(ext, root string, paths []string) ([]string, error) {
	var files []string
	for _, path := range paths {
		err := filepath.WalkDir(filepath.Join(root, path), func(path string, entry os.DirEntry, err error) error {
			if err == nil && !entry.IsDir() && filepath.Ext(path) == ext {
				files = append(files, path)
			}
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func getSchemaFiles(root string, dbVersion db.Version, schemaVersion db.Version) ([]string, error) {
	files, err := getFiles(".sql", root, []string{"ddl", "dml"})
	if err != nil {
		return files, err
	}

	path := filepath.Join(root, "migration")
	migrations, err := os.ReadDir(path)
	if err != nil {
		return files, err
	}

	for _, migration := range migrations {
		if migration.IsDir() {
			migrationPath := filepath.Join(path, migration.Name())

			migrationVersion, err := db.Parse(migration.Name())
			if err != nil {
				color.Printlnf(color.Red, "Invalid migration directory: %s", migrationPath)
				return files, err
			}

			// compare the migration subfolder version with db & schema version
			if dbVersion.GreaterThanOrEq(migrationVersion) || migrationVersion.GreaterThan(schemaVersion) {
				color.Printlnf(color.Yellow, "Skipping migration directory: %s", migrationPath)
			}

			upPath := filepath.Join(migrationPath, "up")

			mFiles, err := os.ReadDir(upPath)
			if err != nil {
				return files, err
			}

			for _, file := range mFiles {
				if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
					files = append(files, filepath.Join(upPath, file.Name()))
				}
			}
		}
	}

	return files, nil
}
