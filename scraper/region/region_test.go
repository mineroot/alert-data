package region_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mineroot/alert-data/scraper/region"
)

func TestParseName(t *testing.T) {
	tests := []struct {
		name     string
		expected region.ID
	}{
		{"м. Київ", region.KyivCity},
		{"Автономна Республіка Крим", region.Crimea},
		{"Івано-Франківська область", region.IvanoFrankivsk},
		{"Курська Народна Республіка", region.Invalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := region.ParseName(test.name)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestParseId(t *testing.T) {
	tests := []struct {
		id       int
		expected region.ID
	}{
		{26, region.KyivCity},
		{1, region.Crimea},
		{9, region.IvanoFrankivsk},
		{-69, region.Invalid},
		{0, region.Invalid},
		{420, region.Invalid},
	}

	for _, test := range tests {
		t.Run(strconv.Itoa(test.id), func(t *testing.T) {
			result := region.ParseId(test.id)
			assert.Equal(t, test.expected, result)
		})
	}
}
