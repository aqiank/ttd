package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

var (
	zolaPath  string // The path to the Zola directory
	dbConnStr string // The database connection string
)

// generateLocationContent generates static-site content for Location page to be used by Zola
func generateLocationContent(location Location) error {
	zolaLocation, err := location.Zola()
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/content/locations/%d.md", zolaPath, location.ID)

	output, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err = output.Write([]byte("+++\n")); err != nil {
		return err
	}

	encoder := toml.NewEncoder(output)
	if err = encoder.Encode(zolaLocation); err != nil {
		return err
	}

	if _, err = output.Write([]byte("+++\n")); err != nil {
		return err
	}

	if _, err = output.Write([]byte(location.Description)); err != nil {
		return err
	}

	// Create the cover image directory in Zola if it doesn't exist
	coverImageDirPath := fmt.Sprintf("%s/static/img/cover/location", zolaPath)
	if err := os.MkdirAll(coverImageDirPath, 0700); err != nil {
		return err
	}

	// Read the cover image file
	coverImageData, err := ioutil.ReadFile("files/" + location.CoverImageURL)
	if err != nil {
		return err
	}

	// Write the cover image file to Zola
	coverImagePath := fmt.Sprintf("%s/static%s", zolaPath, zolaLocation.Extra.CoverImageURL)
	if err = ioutil.WriteFile(coverImagePath, coverImageData, 0600); err != nil {
		return err
	}

	// Create the images directory in Zola if it doesn't exist
	imagesDirPath := fmt.Sprintf("%s/static/img/location/%d", zolaPath, location.ID)
	if err := os.MkdirAll(imagesDirPath, 0700); err != nil {
		return err
	}

	for i, imageURL := range location.ImageURLs {
		imageData, err := ioutil.ReadFile("files/" + imageURL)
		if err != nil {
			return err
		}

		// Copy the image file
		imagePath := fmt.Sprintf("%s/static%s", zolaPath, zolaLocation.Extra.ImageURLs[i])
		if err = ioutil.WriteFile(imagePath, imageData, 0600); err != nil {
			return err
		}
	}

	return nil
}

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
