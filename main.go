package main

import (
	"database/sql"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

var (
	zolaPath  string // The path to the Zola directory
	dbConnStr string // The database connection string
)

func dbConn() (db *sql.DB, err error) {
	db, err = sql.Open("postgres", dbConnStr)
	return
}

func main() {
	dbConnStr = os.Getenv("DATABASE_URL")

	levelStr := os.Getenv("LOGLEVEL")
	if levelStr == "" {
		levelStr = "ERROR"
	}

	level, err := log.ParseLevel(levelStr)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Value:   false,
				Usage:   "Output verbose logs",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "serve administration website",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "host",
						Usage: "set the server host",
						Value: "127.0.0.1",
					},
					&cli.StringFlag{
						Name:  "port",
						Usage: "set the server port",
						Value: "5000",
					},
					&cli.StringFlag{
						Name:        "zola-path",
						Value:       "",
						Destination: &zolaPath,
						Usage:       "Set the path where Zola files will be located",
					},
				},
				Action: serveAPI,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
