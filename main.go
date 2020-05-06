package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	_ "github.com/lib/pq"
)

// CommonData is just a one-off structure for retrieving "type" from an Item's data
type CommonData struct {
	Type string `json:"type"`
}

// ZolaLocation is the format of Location used by Zola
type ZolaLocation struct {
	Title string `toml:"title"`
	Extra struct {
		Description   string                           `toml:"-"`
		Address       string                           `toml:"address"`
		Coordinates   []float64                        `toml:"coordinates"`
		Phone         string                           `toml:"phone"`
		WebsiteURL    string                           `toml:"website_url"`
		CoverImageURL string                           `toml:"cover_image_url"`
		ImageURLs     []string                         `toml:"image_urls"`
		Tags          []string                         `toml:"tags"`
		OpeningHours  map[string][]LocationOpeningHour `toml:"opening_hours"`
	} `toml:"extra"`
}

// Location is a structure that stores location information.
// For example, a store name, address, opening hours, phone number,
// website URL, etc..
type Location struct {
	Title         string            `toml:"title"`
	Description   string            `toml:"description"`
	Address       string            `toml:"address"`
	Coordinates   []float64         `toml:"coordinates"`
	Phone         string            `toml:"phone"`
	WebsiteURL    string            `toml:"website_url",json:"websiteURL"`
	CoverImageURL string            `toml:"cover_image_url",json:"coverImageURL"`
	ImageURLs     []string          `toml:"image_urls",json:"imageURLs"`
	Tags          []string          `toml:"tags"`
	OpeningHours  map[string]string `toml:"opening_hours",json:"openingHours"`
}

// LocationFromData converts a JSONB byte-array into a Location structure
func LocationFromData(data []byte) (location Location, err error) {
	err = json.Unmarshal(data, &location)
	return
}

// Zola converts native format of Location into ZolaLocation
func (location *Location) Zola() (zolaLocation ZolaLocation, err error) {
	zolaLocation.Title = location.Title
	zolaLocation.Extra.Description = location.Description
	zolaLocation.Extra.Address = location.Address
	zolaLocation.Extra.Coordinates = location.Coordinates
	zolaLocation.Extra.Phone = location.Phone
	zolaLocation.Extra.WebsiteURL = location.WebsiteURL

	// Append base path for images to be loaded by Zola
	zolaLocation.Extra.CoverImageURL = fmt.Sprintf("/img/cover/location/%s.jpg", location.CoverImageURL)
	for _, imageURL := range location.ImageURLs {
		zolaLocation.Extra.ImageURLs = append(zolaLocation.Extra.ImageURLs, fmt.Sprintf("/img/location/%s/%s.jpg", location.Title, imageURL))
	}

	zolaLocation.Extra.Tags = location.Tags
	zolaLocation.Extra.OpeningHours, err = location.ZolaOpeningHours()
	return
}

// ZolaOpeningHours converts the OpeningHours representation in Location into the Zola counterpart
func (location *Location) ZolaOpeningHours() (m map[string][]LocationOpeningHour, err error) {
	m = make(map[string][]LocationOpeningHour)

	for k, v := range location.OpeningHours {
		if m[k], err = parseOpeningHours(v); err != nil {
			return
		}
	}

	return
}

// LocationOpeningHour is structure used to store the opening time range of a locaton
//
// For example, a store that opens between 9.30AM to 5 PM would have data such as this:
// LocationOpeningHour {
//     Start: []int{9, 30}
//     End:   []int{17, 0}
// }
//
// A store can also open until after midnight. For example 9.30AM to 2AM would have data such as this:
// LocationOpeningHour {
//     Start: []int{9, 30}
//     End:   []int{26, 0}
// }
type LocationOpeningHour struct {
	Start []int `toml:"start"` // first value is hour, second value is minutes
	End   []int `toml:"end"`
}

var (
	zolaPath string // The path to the Zola directory
)

func parseCoordinates(s string) (coordinates []float64, err error) {
	tokens := strings.Split(s, ",")
	if len(tokens) < 2 {
		err = errors.New("Must provide two numbers separated by command as coordinates")
		return
	}

	var latitude float64
	var longitude float64

	latitude, err = strconv.ParseFloat(tokens[0], 64)
	if err != nil {
		return
	}

	longitude, err = strconv.ParseFloat(tokens[1], 64)
	if err != nil {
		return
	}

	coordinates = append(coordinates, latitude)
	coordinates = append(coordinates, longitude)

	return
}

func parseTags(s string) (tags []string, err error) {
	tags = strings.Split(s, ",")
	return
}

func parseOpeningHoursMap(s string) (openingHoursMap map[string][]LocationOpeningHour, err error) {
	tokens := strings.Split(s, "\n")
	if len(tokens) < 7 {
		err = errors.New("Must provide opening hours for all 7 days")
		return
	}

	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	openingHoursMap = make(map[string][]LocationOpeningHour)

	for i, token := range tokens {
		openingHoursMap[days[i]], err = parseOpeningHours(token)
		if err != nil {
			return
		}
	}

	return
}

// 11-14,17-21
func parseOpeningHours(s string) (openingHours []LocationOpeningHour, err error) {
	tokens := strings.Split(s, ",")

	for _, token := range tokens {
		var openingHour LocationOpeningHour

		if openingHour, err = parseOpeningHour(token); err != nil {
			return
		}

		openingHours = append(openingHours, openingHour)
	}

	return
}

// 11-14
func parseOpeningHour(s string) (openingHour LocationOpeningHour, err error) {
	tokens := strings.Split(s, "-")

	if len(tokens) < 2 {
		err = errors.New("Opening hour must be a range")
		return
	}

	start := tokens[0]
	end := tokens[1]

	startTokens := strings.Split(start, ".")
	endTokens := strings.Split(end, ".")

	var startHour, startMinute, endHour, endMinute int

	startHour, err = strconv.Atoi(startTokens[0])
	if err != nil {
		return
	}

	endHour, err = strconv.Atoi(endTokens[0])
	if err != nil {
		return
	}

	if len(startTokens) > 1 {
		startMinute, err = strconv.Atoi(startTokens[1])
		if err != nil {
			return
		}
	}

	if len(endTokens) > 1 {
		endMinute, err = strconv.Atoi(endTokens[1])
		if err != nil {
			return
		}
	}

	if startHour < 0 || startHour > 24 {
		err = errors.New("Invalid starting hour")
		return
	}

	if endHour < 0 || endHour > 48 {
		err = errors.New("Invalid ending hour")
		return
	}

	if endHour < startHour {
		err = errors.New("Ending hour cannot be before starting hour")
		return
	}

	openingHour.Start = append(openingHour.Start, startHour)
	openingHour.Start = append(openingHour.Start, startMinute)
	openingHour.End = append(openingHour.End, endHour)
	openingHour.End = append(openingHour.End, endMinute)

	return
}

// generateLocationContent generates static-site content for Location page to be used by Zola
func generateLocationContent(location Location) error {
	zolaLocation, err := location.Zola()
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/content/locations/%s.md", zolaPath, location.Title)

	output, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err = output.Write([]byte("+++\n")); err != nil {
		return err
	}

	encoder := toml.NewEncoder(output)
	err = encoder.Encode(zolaLocation)
	if err != nil {
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
	imagesDirPath := fmt.Sprintf("%s/static/img/location/%s", zolaPath, location.Title)
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
	db, err = sql.Open("postgres", "user=ttd dbname=ttd password=abc123 sslmode=disable")
	return
}

func main() {
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
				Action: func(c *cli.Context) error {
					r := gin.Default()

					r.Use(cors.New(cors.Config{
						AllowOrigins:     []string{"*"},
						AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
						AllowHeaders:     []string{"Origin", "Content-Type"},
						ExposeHeaders:    []string{"Content-Length"},
						AllowCredentials: false,
						AllowOriginFunc: func(origin string) bool {
							return origin == "http://localhost:8000"
						},
						MaxAge: 12 * time.Hour,
					}))

					// Get multiple items (events or locations)
					r.GET("/items", func(c *gin.Context) {
						var size int = 10

						sizeStr := c.Query("size")
						if sizeStr != "" {
							var err error

							size, err = strconv.Atoi(sizeStr)
							if err != nil {
								log.Error(err)
								c.JSON(400, gin.H{
									"status":  "error",
									"message": "size is not a valid number",
								})
								return
							}
						}

						if size < 0 {
							size = 1
						}

						db, err := dbConn()
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not connect to the database",
							})
							return
						}
						defer db.Close()

						rows, err := db.Query("SELECT id, data, created_at, updated_at FROM items LIMIT $1", size)
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not execute query",
							})
							return
						}
						defer rows.Close()

						items := make([]Item, 0)

						for rows.Next() {
							var item Item

							if err = rows.Scan(
								&item.ID,
								&item.Data,
								&item.CreatedAt,
								&item.UpdatedAt,
							); err != nil {
								log.Error(err)
								c.JSON(500, gin.H{
									"status":  "error",
									"message": "could not fetch an item",
								})
								return
							}

							items = append(items, item)
						}

						c.JSON(200, items)
					})

					// Get a single item (event or location)
					r.GET("/item/:id", func(c *gin.Context) {
						id, err := strconv.Atoi(c.Param("id"))
						if err != nil {
							log.Error(err)
							c.JSON(400, gin.H{
								"status":  "error",
								"message": errors.New("ID is not valid"),
							})
							return
						}

						db, err := dbConn()
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not connect to the database",
							})
							return
						}
						defer db.Close()

						var item Item

						if err := db.QueryRow("SELECT id, data, created_at, updated_at FROM items WHERE id = $1", id).Scan(
							&item.ID,
							&item.Data,
							&item.CreatedAt,
							&item.UpdatedAt,
						); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not fetch an item",
							})
							return
						}

						var commonData CommonData

						if err = json.Unmarshal(item.Data, &commonData); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not parse item data as common data structure",
							})
							return
						}

						switch commonData.Type {
						case "location":
							var location Location

							if err = json.Unmarshal(item.Data, &location); err != nil {
								log.Error(err)
								c.JSON(500, gin.H{
									"status":  "error",
									"message": "could not parse item data as Location structure",
								})
								return
							}

							c.JSON(200, location)
						default:
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "unknown item type",
							})
						}
					})

					// Create a new item (event or location)
					r.POST("/item", func(c *gin.Context) {
						var data map[string]interface{}

						if err := c.ShouldBindJSON(&data); err != nil {
							c.JSON(400, gin.H{
								"status":  "error",
								"message": errors.New("could not parse JSON in the request"),
							})
							return
						}

						db, err := dbConn()
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not connect to the database",
							})
							return
						}
						defer db.Close()

						// Check for images and store them as files
						if err = storeImages(data); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not store images to the filesystem",
							})
							return
						}

						dataBytes, err := json.Marshal(data)
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not marshal JSON value",
							})
							return
						}

						if _, err := db.Exec("INSERT INTO items (data, created_at, updated_at) VALUES ($1, NOW(), NOW())", dataBytes); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not insert item to database",
							})
							return
						}

						c.JSON(200, gin.H{
							"status":  "ok",
							"message": "successfully created a new item",
						})
					})

					// Update an existing item (event or location)
					r.PUT("/item/:id", func(c *gin.Context) {
						var data map[string]interface{}

						id, err := strconv.Atoi(c.Param("id"))
						if err != nil {
							log.Error(err)
							c.JSON(400, gin.H{
								"status":  "error",
								"message": errors.New("ID is not valid"),
							})
							return
						}

						if err := c.ShouldBindJSON(&data); err != nil {
							c.JSON(400, gin.H{
								"status":  "error",
								"message": errors.New("could not parse JSON in the request"),
							})
							return
						}

						db, err := dbConn()
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not connect to the database",
							})
							return
						}
						defer db.Close()

						// Check for images and store them as files
						if err = storeImages(data); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not store images to the filesystem",
							})
							return
						}

						dataBytes, err := json.Marshal(data)
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not marshal JSON value",
							})
							return
						}

						if _, err := db.Exec("UPDATE items SET data = $1, updated_at = NOW() WHERE id = $2", dataBytes, id); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not update item in the database",
							})
							return
						}

						c.JSON(200, gin.H{
							"status":  "ok",
							"message": "successfully updated item",
						})
					})

					// DELETE an existing item (event or location)
					r.DELETE("/item/:id", func(c *gin.Context) {
						id, err := strconv.Atoi(c.Param("id"))
						if err != nil {
							log.Error(err)
							c.JSON(400, gin.H{
								"status":  "error",
								"message": errors.New("ID is not valid"),
							})
							return
						}

						db, err := dbConn()
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not connect to the database",
							})
							return
						}
						defer db.Close()

						if _, err := db.Exec("DELETE FROM items WHERE id = $1", id); err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not delete item from the database",
							})
							return
						}

						c.JSON(200, gin.H{
							"status":  "ok",
							"message": "successfully deleted item",
						})
					})

					// Run the static site content generator
					r.POST("/generate/:typ", func(c *gin.Context) {
						typ := c.Param("typ")

						db, err := dbConn()
						if err != nil {
							log.Error(err)
							c.JSON(500, gin.H{
								"status":  "error",
								"message": "could not connect to the database",
							})
							return
						}
						defer db.Close()

						rows, err := db.Query("SELECT id, data, created_at, updated_at FROM items WHERE data->>'type' = $1", typ)
						if err != nil {
							return
						}
						defer rows.Close()

						var items []Item

						for rows.Next() {
							var item Item

							if err = rows.Scan(
								&item.ID,
								&item.Data,
								&item.CreatedAt,
								&item.UpdatedAt,
							); err != nil {
								log.Error(err)
								c.JSON(500, gin.H{
									"status":  "error",
									"message": "could not fetch an item",
								})
								return
							}

							items = append(items, item)
						}

						for _, item := range items {
							var commonData CommonData

							if err := json.Unmarshal(item.Data, &commonData); err != nil {
								continue
							}

							switch commonData.Type {
							case "location":
								location, err := LocationFromData(item.Data)
								if err != nil {
									log.Error(err)
									c.JSON(500, gin.H{
										"status":  "error",
										"message": "could not convert internal JSON into location structure",
									})
									return
								}

								if err := generateLocationContent(location); err != nil {
									log.Error(err)
									c.JSON(500, gin.H{
										"status":  "error",
										"message": "could not generate location content",
									})
									return
								}
							default:
								msg := fmt.Sprintf("Item of type \"%s\" is not supported for content generation", typ)
								log.Warn(msg)
								c.JSON(500, gin.H{
									"status":  "error",
									"message": msg,
								})
							}
						}

						c.JSON(200, gin.H{
							"status":  "ok",
							"message": "successfully generated content",
						})
					})

					// Dummy cover image endpoint
					r.POST("/cover", func(c *gin.Context) {
						c.JSON(200, gin.H{
							"status":  "ok",
							"message": "successfully uploaded cover image",
						})
					})

					log.Info("Content will be generated at \"", c.String("zola-path"), "\"")

					r.Run(c.String("host") + ":" + c.String("port"))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
