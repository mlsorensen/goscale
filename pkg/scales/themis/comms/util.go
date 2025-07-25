package comms

import (
	"sort"
	"sync"
)

type AutoOffSetting uint8

const (
	AutoOff5Min  AutoOffSetting = 5 // Auto-off after 5 minutes
	AutoOff10Min AutoOffSetting = 10
	AutoOff15Min AutoOffSetting = 15 // Auto-off after 10 minutes
	AutoOff20Min AutoOffSetting = 20 // Auto-off after 20 minutes
	AutoOff30Min AutoOffSetting = 30 // Auto-off after 30 minutes
)

// autoOffSettingsManager manages a slice of AutoOffSetting values and provides thread-safe access to cycle through them.
type autoOffSettingsManager struct {
	settings       []AutoOffSetting
	sortedSettings []int
	index          int
	mutex          sync.RWMutex
}

// NewAutoOffSettingsManager creates a new instance of autoOffSettingsManager with the given settings.
func newAutoOffSettingsManager() *autoOffSettingsManager {
	settings := []AutoOffSetting{
		AutoOff5Min,
		AutoOff10Min,
		AutoOff15Min,
		AutoOff20Min,
		AutoOff30Min,
	}

	// Create a copy and sort it for binary search purposes
	sortedSettings := make([]int, len(settings))
	for i, setting := range settings {
		sortedSettings[i] = int(setting)
	}
	sort.Slice(sortedSettings, func(i, j int) bool {
		return sortedSettings[i] < sortedSettings[j]
	})

	return &autoOffSettingsManager{
		settings:       settings,
		sortedSettings: sortedSettings,
		index:          0,
		mutex:          sync.RWMutex{},
	}
}

// Next returns the next value in the settings slice, cycling from the beginning after reaching the end.
// It is safe for concurrent access by multiple goroutines.
func (m *autoOffSettingsManager) Next() AutoOffSetting {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	current := m.settings[m.index]
	m.index = (m.index + 1) % len(m.settings)

	return current
}

// NextWithInt finds the next highest setting.
// It uses binary search on a sorted slice of settings for efficient lookup.
// If the input is greater than or equal to the highest setting, it rolls over
// and returns the lowest available setting.
func (m *autoOffSettingsManager) NextWithInt(n uint16) AutoOffSetting {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// This check prevents a panic if the settings slice is ever empty.
	if len(m.sortedSettings) == 0 {
		// 5 minutes is a sensible default
		return 5
	}

	target := int(n)

	// Perform binary search to find the first setting STRICTLY GREATER than the target.
	idx := sort.Search(len(m.sortedSettings), func(i int) bool {
		return m.sortedSettings[i] > target
	})

	// If the index is within the slice, we found the next highest setting.
	if idx < len(m.sortedSettings) {
		return AutoOffSetting(m.sortedSettings[idx])
	}

	// Otherwise, n was >= the max setting, so we roll over to the very first
	// setting in the list (which is now AutoOff1Min).
	return AutoOffSetting(m.sortedSettings[0])
}

// CalculateChecksum computes the checksum by XORing all bytes in the given slice.
func CalculateChecksum(data []byte) byte {
	var checksum byte = 0
	for _, b := range data {
		checksum ^= b
	}
	return checksum
}
