package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"crypto/sha1"

	"github.com/BurntSushi/toml"
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

// generateEventContent generates static-site content for Event page to be used by Zola
func generateEventContent(event Event) error {
	zolaEvent, err := event.Zola()
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/content/events/%d.md", zolaPath, event.ID)

	output, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err = output.Write([]byte("+++\n")); err != nil {
		return err
	}

	encoder := toml.NewEncoder(output)
	if err = encoder.Encode(zolaEvent); err != nil {
		return err
	}

	if _, err = output.Write([]byte("+++\n")); err != nil {
		return err
	}

	if _, err = output.Write([]byte(event.Description)); err != nil {
		return err
	}

	// Create the cover image directory in Zola if it doesn't exist
	coverImageDirPath := fmt.Sprintf("%s/static/img/cover/event", zolaPath)
	if err := os.MkdirAll(coverImageDirPath, 0700); err != nil {
		return err
	}

	// Read the cover image file
	coverImageData, err := ioutil.ReadFile("files/" + event.CoverImageURL)
	if err != nil {
		return err
	}

	// Write the cover image file to Zola
	coverImagePath := fmt.Sprintf("%s/static%s", zolaPath, zolaEvent.Extra.CoverImageURL)
	if err = ioutil.WriteFile(coverImagePath, coverImageData, 0600); err != nil {
		return err
	}

	// Create the images directory in Zola if it doesn't exist
	imagesDirPath := fmt.Sprintf("%s/static/img/event/%d", zolaPath, event.ID)
	if err := os.MkdirAll(imagesDirPath, 0700); err != nil {
		return err
	}

	for i, imageURL := range event.ImageURLs {
		imageData, err := ioutil.ReadFile("files/" + imageURL)
		if err != nil {
			return err
		}

		// Copy the image file
		imagePath := fmt.Sprintf("%s/static%s", zolaPath, zolaEvent.Extra.ImageURLs[i])
		if err = ioutil.WriteFile(imagePath, imageData, 0600); err != nil {
			return err
		}
	}

	return nil
}

// storeImages stores base64 images from data into the filesystem
// and replaces the image data with SHA1 checksum in the map
func storeImages(data map[string]interface{}) (err error) {
	var imageHash string

	for k, v := range data {
		if vv, ok := v.(string); ok && k == "coverImageURL" {
			if imageHash, err = storeImage(data, k, vv); err != nil {
				return
			}

			data[k] = imageHash
		} else if vs, ok := v.([]interface{}); ok && k == "imageURLs" {
			var imageHashes []string
			var imageHashesM = make(map[string]bool)

			for _, v := range vs {
				if vv, ok := v.(string); ok {
					if imageHash, err = storeImage(data, k, vv); err != nil {
						return
					}

					if _, ok = imageHashesM[imageHash]; ok {
						continue
					}

					imageHashes = append(imageHashes, imageHash)
				}
			}

			data[k] = imageHashes
		}
	}

	return
}

// Store an base64 image
func storeImage(data map[string]interface{}, k, v string) (imageHash string, err error) {
	if strings.HasPrefix(v, "data:") {
		i := strings.Index(v, ";base64,")
		if i == 0 && i+len(";base64,") < len(v) {
			return
		}

		hash := sha1.New()
		base64Data := v[i+len(";base64,"):]
		hash.Write([]byte(base64Data))

		if err = os.MkdirAll("files", 0700); err != nil {
			return
		}

		var imageData []byte
		if imageData, err = base64.StdEncoding.DecodeString(base64Data); err != nil {
			return
		}

		imageHash = base64.StdEncoding.EncodeToString(hash.Sum(nil))
		imageHash = strings.ReplaceAll(imageHash, "/", "_")

		filename := "files/" + imageHash
		if _, err = os.Stat(filename); err != nil {
			if err == os.ErrExist {
				return
			}
			err = nil
		}

		if err = ioutil.WriteFile(filename, imageData, 0600); err != nil {
			return
		}

		return
	} else {
		imageHash = v
	}

	return
}

// parseCoordinates parse a string that consists of latitude and longitude separated comma
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

// parseTags parse a list of tags separated by comma
func parseTags(s string) (tags []string, err error) {
	tags = strings.Split(s, ",")
	return
}

// parseOpeningHours parse a list of opening hour ranges string separated by command
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

// parseOpeningHour parse a single opening hour range
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
