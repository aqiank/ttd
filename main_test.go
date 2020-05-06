package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOpeningHoursWithoutMinutes(t *testing.T) {
	s := "7-15,19-28"

	openingHours, err := parseOpeningHours(s)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, openingHours[0].Start[0], 7)
	assert.Equal(t, openingHours[0].End[0], 15)
	assert.Equal(t, openingHours[0].Start[1], 0)
	assert.Equal(t, openingHours[0].End[1], 0)

	assert.Equal(t, openingHours[1].Start[0], 19)
	assert.Equal(t, openingHours[1].End[0], 28)
	assert.Equal(t, openingHours[1].Start[1], 0)
	assert.Equal(t, openingHours[1].End[1], 0)
}

func TestParseOpeningHoursWithPartialMinutes(t *testing.T) {
	s := "7-15.30,19.30-28"

	openingHours, err := parseOpeningHours(s)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, openingHours[0].Start[0], 7)
	assert.Equal(t, openingHours[0].End[0], 15)
	assert.Equal(t, openingHours[0].Start[1], 0)
	assert.Equal(t, openingHours[0].End[1], 30)

	assert.Equal(t, openingHours[1].Start[0], 19)
	assert.Equal(t, openingHours[1].End[0], 28)
	assert.Equal(t, openingHours[1].Start[1], 30)
	assert.Equal(t, openingHours[1].End[1], 0)
}

func TestParseOpeningHoursWithMinutes(t *testing.T) {
	s := "7.30-15.30,19.30-28.00"

	openingHours, err := parseOpeningHours(s)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, openingHours[0].Start[0], 7)
	assert.Equal(t, openingHours[0].End[0], 15)
	assert.Equal(t, openingHours[0].Start[1], 30)
	assert.Equal(t, openingHours[0].End[1], 30)

	assert.Equal(t, openingHours[1].Start[0], 19)
	assert.Equal(t, openingHours[1].End[0], 28)
	assert.Equal(t, openingHours[1].Start[1], 30)
	assert.Equal(t, openingHours[1].End[1], 0)
}

func TestParseOpeningHour(t *testing.T) {
	s := "7.30-15.30"

	openingHour, err := parseOpeningHour(s)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, openingHour.Start[0], 7)
	assert.Equal(t, openingHour.End[0], 15)
	assert.Equal(t, openingHour.Start[1], 30)
	assert.Equal(t, openingHour.End[1], 30)
}
