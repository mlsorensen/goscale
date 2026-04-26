package goscale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

// FoundDevice struct remains the same.
type FoundDevice struct {
	Name    string
	Address bluetooth.Address
	RSSI    int
}

var BTAdapter = bluetooth.DefaultAdapter

// ScanForOne scans until the first registered scale name is found
func ScanForOne(duration time.Duration) (*FoundDevice, error) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	err := TryEnableAdapter()
	if err != nil {
		return nil, err
	}

	var found FoundDevice
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
				log.Printf("    --> Found a match! Device: %s", name)
				found = FoundDevice{
					Name:    name,
					Address: result.Address,
					RSSI:    int(result.RSSI),
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
		err := BTAdapter.Scan(handler)
		if err != nil {
			scanErrChan <- err
			// Wake the main goroutine immediately rather than waiting for
			// the scan timeout. Without this we'd sit silently until ctx
			// fires.
			cancel()
		}
	}()

	<-ctx.Done()

	log.Println("Stopping scan...")
	err = BTAdapter.StopScan()
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
	err := TryEnableAdapter()
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
						Name:    name,
						Address: result.Address,
						RSSI:    int(result.RSSI),
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
		err := BTAdapter.Scan(handler)
		if err != nil {
			scanErrChan <- err
			// Wake the main goroutine immediately rather than waiting for
			// the scan timeout. Without this we'd sit silently until ctx
			// fires.
			cancel()
		}
	}()

	<-ctx.Done()

	log.Println("Timeout reached. Stopping scan...")
	err = BTAdapter.StopScan()
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

// ScanAndConnect scans for any registered scale, looks up the matching
// implementation, and connects. Returns the live Scale and its weight-update
// channel on success. Equivalent to ScanForOne + NewScaleForDevice +
// Scale.Connect — packaged because that's the typical "find and connect to
// the first available scale" call.
func ScanAndConnect(scanTimeout time.Duration) (Scale, <-chan WeightUpdate, error) {
	dev, err := ScanForOne(scanTimeout)
	if err != nil {
		return nil, nil, err
	}
	if dev == nil || dev.Name == "" {
		return nil, nil, errors.New("scan: no scale found")
	}

	s, err := NewScaleForDevice(dev)
	if err != nil {
		return nil, nil, fmt.Errorf("scan: %w", err)
	}

	updates, err := s.Connect()
	if err != nil {
		return nil, nil, fmt.Errorf("scan: connect: %w", err)
	}

	return s, updates, nil
}

func TryEnableAdapter() error {
	log.Println("Enabling Bluetooth BTAdapter...")
	err := BTAdapter.Enable()
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
