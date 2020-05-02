package main

import "time"

type Item struct {
	ID        int64     `json:"id"`
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
