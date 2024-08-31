package region_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mineroot/alert-data/scraper/region"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		expected region.ID
		hasError bool
	}{
		{"м. Київ", region.KyivCity, false},
		{"Автономна Республіка Крим", region.Crimea, false},
		{"Івано-Франківська область", region.IvanoFrankivsk, false},
		{"Курська Народна Республіка", region.Invalid, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := region.Parse(test.name)
			if test.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expected, result)
		})
	}
}
