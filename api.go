package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// getItems fetches multiple items (events or locations) from database
func getItems(c *gin.Context) {
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

	items := make([]DecodedItem, 0)

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

		decodedItem, err := item.Decode()
		if err != nil {
			log.Error(err)
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "could not decode an item",
			})
			return
		}

		items = append(items, decodedItem)
	}

	c.JSON(200, items)
}

// getItem fetches a single item (event or location) from database
func getItem(c *gin.Context) {
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

	var itemCommonData ItemCommonData

	if err = json.Unmarshal(item.Data, &itemCommonData); err != nil {
		log.Error(err)
		c.JSON(500, gin.H{
			"status":  "error",
			"message": "could not parse item data as common data structure",
		})
		return
	}

	switch itemCommonData.Type {
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

		location.ID = item.ID
		location.CreatedAt = item.CreatedAt
		location.UpdatedAt = item.UpdatedAt

		c.JSON(200, location)
	case "event":
		var event Event

		if err = json.Unmarshal(item.Data, &event); err != nil {
			log.Error(err)
			c.JSON(500, gin.H{
				"status":  "error",
				"message": "could not parse item data as Event structure",
			})
			return
		}

		event.ID = item.ID
		event.CreatedAt = item.CreatedAt
		event.UpdatedAt = item.UpdatedAt

		c.JSON(200, event)
	default:
		c.JSON(500, gin.H{
			"status":  "error",
			"message": "unknown item type",
		})
	}
}

// postItem creates a new item (event or location) in database
func postItem(c *gin.Context) {
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
}

// putItem updates an existing item (event or location) in the database
func putItem(c *gin.Context) {
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
}

// deleteItem deletes an item (event or location) in the database
func deleteItem(c *gin.Context) {
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
}

// postGenerate generates markdown pages and places assets in the Zola directory
func postGenerate(c *gin.Context) {
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
		var itemCommonData ItemCommonData

		if err := json.Unmarshal(item.Data, &itemCommonData); err != nil {
			continue
		}

		switch itemCommonData.Type {
		case "location":
			location, err := LocationFromItem(item)
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
		case "event":
			event, err := EventFromItem(item)
			if err != nil {
				log.Error(err)
				c.JSON(500, gin.H{
					"status":  "error",
					"message": "could not convert internal JSON into event structure",
				})
				return
			}

			if err := generateEventContent(event); err != nil {
				log.Error(err)
				c.JSON(500, gin.H{
					"status":  "error",
					"message": "could not generate event content",
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
}

func serveAPI(c *cli.Context) error {
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

	r.GET("/items", getItems)

	// Get a single item (event or location)
	r.GET("/item/:id", getItem)

	// Create a new item (event or location)
	r.POST("/item", postItem)

	// Update an existing item (event or location)
	r.PUT("/item/:id", putItem)

	// Delete an existing item (event or location)
	r.DELETE("/item/:id", deleteItem)

	// Run the static site content generator
	r.POST("/generate/:typ", postGenerate)

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
}
