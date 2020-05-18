package main

import (
	"encoding/json"
	"time"
)

type Item struct {
	ID        int64     `json:"id"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type DecodedItem struct {
	ID        int64                  `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// Decode turns Item into DecodedItem
func (item *Item) Decode() (decodedItem DecodedItem, err error) {
	decodedItem.ID = item.ID

	if err = json.Unmarshal(item.Data, &decodedItem.Data); err != nil {
		return
	}

	decodedItem.CreatedAt = item.CreatedAt
	decodedItem.UpdatedAt = item.UpdatedAt
	return
}

// ItemCommonData is just a one-off structure for retrieving "type" from an Item's data
type ItemCommonData struct {
	Type string `json:"type"`
}
