package scraper

import (
	"fmt"
	"sync"
	"time"

	"github.com/mineroot/alert-data/scraper/region"
)

// Status represents the alert status for a region.
type Status struct {
	Region    region.ID
	Enabled   bool
	UpdatedAt time.Time
	IsHistory bool // if this is true UpdatedAt may be inaccurate (zero)
}

// AlertData holds the raid status information for all regions.
type AlertData struct {
	lock *sync.RWMutex
	data map[region.ID]*Status
}

func newAlertData() *AlertData {
	alertData := &AlertData{
		lock: &sync.RWMutex{},
		data: make(map[region.ID]*Status, region.Count()),
	}

	// assume raid alert is disabled for all regions
	for id, _ := range region.Iterator() {
		alertData.set(&Status{
			Region:    id,
			Enabled:   false,
			UpdatedAt: time.Time{},
			IsHistory: true,
		})
	}

	// hardcode raid alerts in Crimea & Luhansk regions
	// as it's long-running, and it's inefficient to parse Tg channel for last 2+ years
	alertData.set(&Status{
		Region:    region.Crimea,
		Enabled:   true,
		UpdatedAt: time.Date(2022, time.December, 11, 0, 22, 0, 0, kyivLocation),
		IsHistory: true,
	})
	alertData.set(&Status{
		Region:    region.Luhansk,
		Enabled:   true,
		UpdatedAt: time.Date(2022, time.April, 4, 19, 45, 0, 0, kyivLocation),
		IsHistory: true,
	})
	return alertData
}

// GetByRegion retrieves the alert status for a specific region.
// Returns an error if the region is invalid.
func (r *AlertData) GetByRegion(id region.ID) (Status, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	currentStatus, exists := r.data[id]
	if !exists {
		return Status{}, fmt.Errorf("scraper: invalid region '%s'", id)
	}
	return *currentStatus, nil
}

func (r *AlertData) set(newStatus *Status) {
	if newStatus == nil {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	currentStatus, exists := r.data[newStatus.Region]
	if exists && newStatus.UpdatedAt.Before(currentStatus.UpdatedAt) {
		// skip update if new status is older than current status
		return
	}
	r.data[newStatus.Region] = newStatus
}
