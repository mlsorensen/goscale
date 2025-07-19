package goscale

import (
	"context"
	"errors"
	"log"
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

// ScanStream returns a channel that streams FoundDevice as they are discovered
// and stops scanning when the context is canceled.
func ScanStream(ctx context.Context, customPrefixes ...string) (<-chan FoundDevice, error) {
	deviceChan := make(chan FoundDevice)

	// Start the scan goroutine
	go func() {
		defer close(deviceChan)

		mu := sync.Mutex{}
		foundDevices := make(map[string]FoundDevice)
		prefixesToScan := getPrefixes(customPrefixes...)

		if len(prefixesToScan) == 0 {
			return // No prefixes to scan for, nothing to do
		}

		log.Printf("Starting BLE scan for devices with prefixes: %v...", prefixesToScan)

		handler := func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			name := result.LocalName()

			if name == "" {
				return // Ignore packets without a name.
			}

			mu.Lock()
			defer mu.Unlock()

			for _, prefix := range prefixesToScan {
				if strings.HasPrefix(name, prefix) {
					deviceChan <- FoundDevice{
						Name: name,
						ID:   result.Address.String(),
						RSSI: int(result.RSSI),
					}

					// Add the device to the foundDevices map
					foundDevices[result.Address.String()] = FoundDevice{
						Name: name,
						ID:   result.Address.String(),
						RSSI: int(result.RSSI),
					}
				}
			}
		}

		err := adapter.Scan(handler)
		if err != nil {
			log.Printf("Error starting scan: %v", err)
			return
		}

		// Wait for the context to be canceled
		<-ctx.Done()

		// Stop the scan and clean up
		if err := adapter.StopScan(); err != nil {
			log.Printf("Error stopping scan: %v", err)
		}
	}()

	return deviceChan, nil
}

// Scan finds any bluetooth devices with given string prefixes in their name, blocks for duration
func Scan(duration time.Duration, customPrefixes ...string) ([]FoundDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	log.Println("Enabling Bluetooth adapter...")
	err := adapter.Enable()
	if err != nil {
		return nil, err
	}

	mu := sync.Mutex{}
	foundDevices := make(map[string]FoundDevice)
	prefixesToScan := getPrefixes(customPrefixes...)

	if len(prefixesToScan) == 0 {
		return nil, errors.New("Scan warning: no implementations registered and no custom prefixes provided.")
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
				mu.Lock()
				id := result.Address.String()
				foundDevices[id] = FoundDevice{
					Name: name,
					ID:   id,
					RSSI: int(result.RSSI),
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
		log.Println("Starting blocking scan...")
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

// getPrefixes helper function, provide prefixes in addition to registered scale prefixes
func getPrefixes(customPrefixes ...string) []string {
	if len(customPrefixes) > 0 {
		return customPrefixes
	}
	regLock.RLock()
	defer regLock.RUnlock()
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	return keys
}
