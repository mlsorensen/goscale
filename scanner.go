package goscale

import (
	"context"
	"errors"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// FoundDevice struct remains the same.
type FoundDevice struct {
	Name string
	ID   string
	RSSI int
}

var adapter = bluetooth.DefaultAdapter

// ScanForOne scans until the first registered scale name is found
func ScanForOne(duration time.Duration) (*FoundDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	err := tryEnableBTAdapter()
	if err != nil {
		return nil, err
	}

	var found FoundDevice
	prefixesToScan := getRegisteredPrefixes()

	if len(prefixesToScan) == 0 {
		return nil, errors.New("scan warning: no implementations registered.")
	}
	log.Printf("Scanning for devices with prefixes: %v.", prefixesToScan)

	handler := func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		name := result.LocalName()

		if name == "" {
			return // Ignore packets without a name.
		}

		for _, prefix := range prefixesToScan {
			if strings.HasPrefix(name, prefix) {
				log.Printf("    --> Found a match! Device: %s", name)
				id := result.Address.String()
				found = FoundDevice{
					Name: name,
					ID:   id,
					RSSI: int(result.RSSI),
				}
				cancel()
				break
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	scanErrChan := make(chan error, 1)

	go func() {
		defer wg.Done()
		log.Println("Starting a blocking scan...")
		err := adapter.Scan(handler)
		if err != nil {
			scanErrChan <- err
		}
	}()

	<-ctx.Done()

	log.Println("Stopping scan...")
	err = adapter.StopScan()
	if err != nil {
		log.Printf("Warning: failed to stop scan cleanly: %v", err)
	}

	wg.Wait()
	close(scanErrChan)

	if scanErr := <-scanErrChan; scanErr != nil {
		return nil, scanErr
	}

	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}

	log.Printf("Scan processing finished. Found matching device %v", &found)
	return &found, nil
}

// Scan finds any bluetooth devices with given string prefixes in their name, blocks for duration
func Scan(duration time.Duration) ([]FoundDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	err := tryEnableBTAdapter()
	if err != nil {
		return nil, err
	}

	mu := sync.Mutex{}
	foundDevices := make(map[string]FoundDevice)
	prefixesToScan := getRegisteredPrefixes()

	if len(prefixesToScan) == 0 {
		return nil, errors.New("scan warning: no implementations registered")
	}
	log.Printf("Scanning for devices with prefixes: %v.", prefixesToScan)

	handler := func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		name := result.LocalName()

		if name == "" {
			return // Ignore packets without a name.
		}

		for _, prefix := range prefixesToScan {
			if strings.HasPrefix(name, prefix) {
				id := result.Address.String()
				mu.Lock()
				if _, exists := foundDevices[id]; !exists {
					log.Printf("    --> Found a match! Device: %s", name)
					foundDevices[id] = FoundDevice{
						Name: name,
						ID:   id,
						RSSI: int(result.RSSI),
					}
				}
				mu.Unlock()
				break
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	scanErrChan := make(chan error, 1)

	go func() {
		defer wg.Done()
		log.Println("Starting a blocking scan...")
		err := adapter.Scan(handler)
		if err != nil {
			scanErrChan <- err
		}
	}()

	<-ctx.Done()

	log.Println("Timeout reached. Stopping scan...")
	err = adapter.StopScan()
	if err != nil {
		log.Printf("Warning: failed to stop scan cleanly: %v", err)
	}

	wg.Wait()
	close(scanErrChan)

	if scanErr := <-scanErrChan; scanErr != nil {
		return nil, scanErr
	}

	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}

	results := make([]FoundDevice, 0, len(foundDevices))
	for _, device := range foundDevices {
		results = append(results, device)
	}

	log.Printf("Scan processing finished. Found %d unique matching device(s).", len(results))
	return results, nil
}

func tryEnableBTAdapter() error {
	log.Println("Enabling Bluetooth adapter...")
	err := adapter.Enable()
	if err == nil || strings.Contains(err.Error(), "already calling Enable") {
		return nil
	}
	return err
}

// getRegisteredPrefixes helper function
// optional customPrefixes allow one to provide prefixes in addition to registered scale prefixes
func getRegisteredPrefixes(customPrefixes ...string) []string {
	if len(customPrefixes) > 0 {
		return customPrefixes
	}
	regLock.RLock()
	defer regLock.RUnlock()
	keys := make([]string, 0, len(registry))
	for k := range registry {
		if !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	return keys
}
