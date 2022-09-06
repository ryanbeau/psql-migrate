package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jackc/pgx/v4"
	"github.com/ryanbeau/psql-migrate/internal/migrate"
	"github.com/ryanbeau/psql-migrate/pkg/color"
	"github.com/urfave/cli/v2"
)

const version = "v0.1.0-dev"

var migrateCmd = &cli.Command{
	Name:  "migrate",
	Usage: "database migration",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "show logs"},
		&cli.StringFlag{Name: "source", Aliases: []string{"s"}, Usage: "root sql directory", Required: true},
		&cli.StringFlag{Name: "database", Aliases: []string{"d"}, Usage: "postgresql connection string", Required: true},
	},
	Action: func(ctx *cli.Context) error {
		source := ctx.String("source")
		database := ctx.String("database")

		conn, err := pgx.Connect(ctx.Context, database)
		if err != nil {
			return err
		}
		defer conn.Close(ctx.Context)

		return migrate.Run(ctx.Context, conn, filepath.Clean(source))
	},
}

var versionCmd = &cli.Command{
	Name:  "version",
	Usage: "print the version string",
	Action: func(ctx *cli.Context) error {
		fmt.Println(version)
		return nil
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "pgsql-migrate"
	app.Usage = "run postgresql database migrations based on schema"
	app.Description = "This is a library for running postgresql migrations in golang. See https://github.com/ryanbeau/psql-migrate for a getting started guide."
	app.HideVersion = true
	app.Version = version
	app.Action = migrateCmd.Action
	app.Flags = migrateCmd.Flags
	app.Before = func(ctx *cli.Context) error {
		if ctx.Bool("verbose") {
			log.SetFlags(0)
		} else {
			log.SetOutput(io.Discard)
		}
		return nil
	}
	app.Commands = []*cli.Command{
		migrateCmd,
		versionCmd,
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-c
		color.Fprintlnf(os.Stderr, color.Yellow, "\rReceived signal: %v. Aborting...", s)
		cancel()
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil && !errors.Is(errors.Unwrap(err), context.Canceled) {
		color.Fprintln(os.Stderr, color.Red, err.Error())
		os.Exit(1)
	}
}
