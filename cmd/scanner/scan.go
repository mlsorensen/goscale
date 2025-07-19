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

	// Use the simple ScanFor helper function. It will block for the specified duration.
	// It will automatically search for any registered prefixes (e.g., "MOCK").
	// Find any device whose name starts with "LUNAR", for example.
	devices, err := goscale.Scan(scanDuration, "LUNAR")
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
			fmt.Printf("   ID:   %s\n", device.ID)
			fmt.Printf("   RSSI: %d\n\n", device.RSSI)
		}
		fmt.Println("-----------------------------")
	}
}
