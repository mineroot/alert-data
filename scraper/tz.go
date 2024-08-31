package scraper

import (
	"fmt"
	"time"
)

var kyivLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		panic(fmt.Errorf("unable to load Europe/Kyiv timezone: %w", err))
	}
	kyivLocation = loc
}
