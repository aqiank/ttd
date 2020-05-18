package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// ZolaEvent is the format of Event used by Zola
type ZolaEvent struct {
	ID         int64  `toml:"id"`
	Title      string `toml:"title"`
	Taxonomies struct {
		Tags []string `toml:"tags"`
	} `toml:"taxonomies"`
	Extra struct {
		Type          string    `toml:"type"`
		Description   string    `toml:"-"`
		Address       string    `toml:"address"`
		Coordinates   []float64 `toml:"coordinates"`
		Phone         string    `toml:"phone"`
		WebsiteURL    string    `toml:"website_url"`
		CoverImageURL string    `toml:"cover_image_url"`
		ImageURLs     []string  `toml:"image_urls"`
	} `toml:"extra"`
	CreatedAt time.Time `toml:"date"`
	UpdatedAt time.Time `toml:"updated_at"`
}

// Event is a structure that stores event information.
// For example, the event name, address, schedule, phone number,
// website URL, etc..
type Event struct {
	ID            int64     `toml:"id"`
	Type          string    `toml:"type"`
	Title         string    `toml:"title"`
	Description   string    `toml:"description"`
	Address       string    `toml:"address"`
	Coordinates   []float64 `toml:"coordinates"`
	Phone         string    `toml:"phone"`
	WebsiteURL    string    `toml:"website_url" json:"websiteURL"`
	CoverImageURL string    `toml:"cover_image_url" json:"coverImageURL"`
	ImageURLs     []string  `toml:"image_urls" json:"imageURLs"`
	Tags          []string  `toml:"tags"`
	CreatedAt     time.Time `toml:"created_at"`
	UpdatedAt     time.Time `toml:"updated_at"`
}

// EventFromData converts a JSONB byte-array into an Event structure
func EventFromData(data []byte) (event Event, err error) {
	err = json.Unmarshal(data, &event)
	return
}

// EventFromItem creates a event out of an Item
func EventFromItem(item Item) (event Event, err error) {
	event, err = EventFromData(item.Data)
	if err != nil {
		return
	}

	event.ID = item.ID
	event.CreatedAt = item.CreatedAt
	event.UpdatedAt = item.UpdatedAt

	return
}

// Zola converts native format of Event into ZolaEvent
func (event *Event) Zola() (zolaEvent ZolaEvent, err error) {
	zolaEvent.ID = event.ID
	zolaEvent.Title = event.Title
	zolaEvent.Extra.Type = event.Type
	zolaEvent.Extra.Description = event.Description
	zolaEvent.Extra.Address = event.Address
	zolaEvent.Extra.Coordinates = event.Coordinates
	zolaEvent.Extra.Phone = event.Phone
	zolaEvent.Extra.WebsiteURL = event.WebsiteURL

	// Append base path for images to be loaded by Zola
	zolaEvent.Extra.CoverImageURL = fmt.Sprintf("/img/cover/event/%s.jpg", event.CoverImageURL)
	for _, imageURL := range event.ImageURLs {
		zolaEvent.Extra.ImageURLs = append(zolaEvent.Extra.ImageURLs, fmt.Sprintf("/img/event/%d/%s.jpg", event.ID, imageURL))
	}

	zolaEvent.Taxonomies.Tags = event.Tags
	zolaEvent.CreatedAt = event.CreatedAt
	zolaEvent.UpdatedAt = event.UpdatedAt
	return
}
