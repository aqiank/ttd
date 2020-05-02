package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOpeningHoursMap(t *testing.T) {
	s :=
		`7.30-15.30,19.30-28.00
7.30-15.30,19.30-28.00
7.30-15.30,19.30-28.00
7.30-15.30,19.30-28.00
7.30-15.30,19.30-28.00
7.30-15.30,19.30-28.00
7.30-15.30,19.30-28.00`

	openingHoursMap, err := parseOpeningHoursMap(s)
	if err != nil {
		t.Error(err)
	}

	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	for openingHoursDay := range openingHoursMap {
		ok := false

		for _, day := range days {
			if openingHoursDay == day {
				ok = true
			}
		}

		if !ok {
			t.Error(openingHoursDay + " is not a valid day")
		}
	}
}

func TestParseOpeningHours(t *testing.T) {
	s := "7.30-15.30,19.30-28.00"

	openingHours, err := parseOpeningHours(s)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, openingHours[0].Start[0], 7, "Starting hour should be 7")
	assert.Equal(t, openingHours[0].End[0], 15, "Ending hour should be 15")
	assert.Equal(t, openingHours[0].Start[1], 30, "Starting minute should be 30")
	assert.Equal(t, openingHours[0].End[1], 30, "Ending minute should be 30")

	assert.Equal(t, openingHours[1].Start[0], 19, "Starting hour should be 19")
	assert.Equal(t, openingHours[1].End[0], 28, "Ending hour should be 28")
	assert.Equal(t, openingHours[1].Start[1], 30, "Starting minute should be 30")
	assert.Equal(t, openingHours[1].End[1], 0, "Ending minute should be 0")
}

func TestParseOpeningHour(t *testing.T) {
	s := "7.30-15.30"

	openingHour, err := parseOpeningHour(s)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, openingHour.Start[0], 7, "Starting hour should be 7")
	assert.Equal(t, openingHour.End[0], 15, "Ending hour should be 15")
	assert.Equal(t, openingHour.Start[1], 30, "Starting minute should be 30")
	assert.Equal(t, openingHour.End[1], 30, "Ending minute should be 30")
}
