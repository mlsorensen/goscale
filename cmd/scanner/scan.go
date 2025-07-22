package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mlsorensen/goscale"
	_ "github.com/mlsorensen/goscale/pkg/scales/all"
)

func main() {
	log.Println("--- GoScale Scanner Test ---")

	scanDuration := 15 * time.Second
	log.Printf("Starting BLE scan for %s...", scanDuration)
	log.Println("Turn on your Bluetooth scale now.")

	// ScanForOne will automatically search for any registered prefixes (e.g., "LUNAR").
	// It will return as soon as one device is found, or timeout duration, if none.
	device, err := goscale.ScanForOne(scanDuration)
	if err != nil {
		log.Fatalf("Fatal: Scan failed: %v", err)
	}

	if device != nil {
		fmt.Println("\n--- Found Supported Device ---")
		fmt.Printf("   Name: %s\n", device.Name)
		fmt.Printf("   ID:   %s\n", device.Address)
		fmt.Printf("   RSSI: %d\n\n", device.RSSI)
		fmt.Println("-----------------------------")
	} else {
		fmt.Println("\n--- Found NO Device ---")
	}

	// Using the Scan helper function. It will block for the specified duration.
	// It will automatically search for any registered prefixes (e.g., "MOCK").
	// It will return all found devices matching registered prefixes
	devices, err := goscale.Scan(scanDuration)
	if err != nil {
		log.Fatalf("Fatal: Scan failed: %v", err)
	}

	// --- Print the results ---
	if len(devices) == 0 {
		log.Println("\nScan complete. No supported devices found.")
		log.Println("Tip: Make sure your device is on, discoverable, and that you have an implementation for it (e.g., 'LUNAR').")
	} else {
		fmt.Println("\n--- Found Supported Devices ---")
		for i, device := range devices {
			fmt.Printf("%d: Name: %s\n", i+1, device.Name)
			fmt.Printf("   ID:   %s\n", device.Address)
			fmt.Printf("   RSSI: %d\n\n", device.RSSI)
		}
		fmt.Println("-----------------------------")
	}
}
