package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ZolaLocation is the format of Location used by Zola
type ZolaLocation struct {
	ID         int64  `toml:"id"`
	Title      string `toml:"title"`
	Taxonomies struct {
		Tags []string `toml:"tags"`
	} `toml:"taxonomies"`
	Extra struct {
		Description   string                           `toml:"-"`
		Address       string                           `toml:"address"`
		Coordinates   []float64                        `toml:"coordinates"`
		Phone         string                           `toml:"phone"`
		WebsiteURL    string                           `toml:"website_url"`
		CoverImageURL string                           `toml:"cover_image_url"`
		ImageURLs     []string                         `toml:"image_urls"`
		OpeningHours  map[string][]LocationOpeningHour `toml:"opening_hours"`
	} `toml:"extra"`
	CreatedAt time.Time `toml:"date"`
	UpdatedAt time.Time `toml:"updated_at"`
}

// Location is a structure that stores location information.
// For example, a store name, address, opening hours, phone number,
// website URL, etc..
type Location struct {
	ID            int64             `toml:"id"`
	Title         string            `toml:"title"`
	Description   string            `toml:"description"`
	Address       string            `toml:"address"`
	Coordinates   []float64         `toml:"coordinates"`
	Phone         string            `toml:"phone"`
	WebsiteURL    string            `toml:"website_url" json:"websiteURL"`
	CoverImageURL string            `toml:"cover_image_url" json:"coverImageURL"`
	ImageURLs     []string          `toml:"image_urls" json:"imageURLs"`
	Tags          []string          `toml:"tags"`
	OpeningHours  map[string]string `toml:"opening_hours" json:"openingHours"`
	CreatedAt     time.Time         `toml:"created_at"`
	UpdatedAt     time.Time         `toml:"updated_at"`
}

// LocationFromData converts a JSONB byte-array into a Location structure
func LocationFromData(data []byte) (location Location, err error) {
	err = json.Unmarshal(data, &location)
	return
}

// LocationFromItem creates a Location out of an Item
func LocationFromItem(item Item) (location Location, err error) {
	location, err = LocationFromData(item.Data)
	if err != nil {
		return
	}

	location.ID = item.ID
	location.CreatedAt = item.CreatedAt
	location.UpdatedAt = item.UpdatedAt

	return
}

// Zola converts native format of Location into ZolaLocation
func (location *Location) Zola() (zolaLocation ZolaLocation, err error) {
	zolaLocation.ID = location.ID
	zolaLocation.Title = location.Title
	zolaLocation.Extra.Description = location.Description
	zolaLocation.Extra.Address = location.Address
	zolaLocation.Extra.Coordinates = location.Coordinates
	zolaLocation.Extra.Phone = location.Phone
	zolaLocation.Extra.WebsiteURL = location.WebsiteURL

	// Append base path for images to be loaded by Zola
	zolaLocation.Extra.CoverImageURL = fmt.Sprintf("/img/cover/location/%s.jpg", location.CoverImageURL)
	for _, imageURL := range location.ImageURLs {
		zolaLocation.Extra.ImageURLs = append(zolaLocation.Extra.ImageURLs, fmt.Sprintf("/img/location/%d/%s.jpg", location.ID, imageURL))
	}

	zolaLocation.Taxonomies.Tags = location.Tags
	zolaLocation.Extra.OpeningHours, err = location.ZolaOpeningHours()
	zolaLocation.CreatedAt = location.CreatedAt
	zolaLocation.UpdatedAt = location.UpdatedAt
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
