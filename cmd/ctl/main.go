package main

import (
	"database/sql"
	"log"
	"math/rand"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/alexedwards/argon2id"
	"github.com/oklog/ulid/v2"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/urfave/cli/v2"
)

func addUser(ctx *cli.Context) error {
	log.Println(ctx.Args().Slice())
	db, err := sql.Open("sqlite", ctx.String("db"))
	if err != nil {
		return err
	}
	stmt, err := db.PrepareContext(ctx.Context, "INSERT INTO users VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	hashedPassword, err := argon2id.CreateHash(ctx.String("password"), argon2id.DefaultParams)
	if err != nil {
		return err
	}
	entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now()
	id, _ := ulid.New(ulid.Timestamp(now), entropy)
	if _, err = stmt.ExecContext(ctx.Context, id.String(), ctx.Args().Get(0), hashedPassword, now.Format(time.RFC3339)); err != nil {
		return err
	}
	return nil
}

func applyMigrate(ctx *cli.Context) error {
	db, err := sql.Open("sqlite", ctx.String("db"))
	if err != nil {
		return err
	}
	migrations := &migrate.FileMigrationSource{
		Dir: ctx.String("migration"),
	}
	n, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return err
	}
	log.Printf("Applied %d migrations!\n", n)
	return nil
}

func main() {
	app := &cli.App{
		Name:  "threadsctl",
		Usage: "manage threads",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "db",
				Value:    "./db.sqlite",
				Usage:    "place of sqlite file",
				EnvVars:  []string{"THREADS_DB"},
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "adduser",
				Usage:  "adds user",
				Action: addUser,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "password",
						Value:    "",
						Usage:    "password",
						Required: true,
					},
				},
			},
			{
				Name:   "migrate",
				Usage:  "migrate sqlite db",
				Action: applyMigrate,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "migration",
						Value: "./db/migrations",
						Usage: "place of migrations file",
					},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
