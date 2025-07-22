package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mlsorensen/goscale"

	// This tells the Go compiler to include the package, which runs its init()
	// function. The init() function, in turn, calls goscale.Register(). You can
	// specify specific scales individually or just "all"
	_ "github.com/mlsorensen/goscale/pkg/scales/all"
)

func main() {
	log.Println("GoScale CLI Application Starting...")

	// To use the mock, we need to request a device name that matches the prefix
	// it was registered with ("MOCK"). In a real program, we would scan for bluetooth
	// scales and then use its device to create a new Scale
	device := &goscale.FoundDevice{Name: "MOCK-Development-Scale"}
	log.Printf("Attempting to create scale instance for device: %v", device)

	// Use the factory to find and create the correct scale implementation.
	// Again, this device name in a real program would likely come from scanning for
	// supported devices or a device with known name.
	myScale, err := goscale.NewScaleForDevice(device)
	if err != nil {
		log.Fatalf("Fatal: Could not create scale instance: %v", err)
	}
	log.Println("Successfully created mock scale instance.")

	// --- Set up graceful shutdown ---
	// This goroutine listens for OS signals (like Ctrl+C).
	// When a signal is caught, it calls cancel() to trigger a clean shutdown.
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		<-sigchan // Block until a signal is received
		log.Println("Shutdown signal received. Disconnecting...")
		_ = myScale.Disconnect()
	}()

	// Connect to the scale. This will return a channel for weight updates.
	weightUpdates, err := myScale.Connect()
	if err != nil {
		// If connect fails, we must exit. We also call cancel() to clean up
		// the signal-handling goroutine.
		_ = myScale.Disconnect()
		log.Fatalf("Fatal: Could not connect to scale: %v", err)
	}
	log.Println("Connection successful. Listening for weight updates...")

	// This goroutine will run in the background to interact with the scale
	// while the main goroutine is busy listening for weight updates.
	go func() {
		for {
			// Wait a few seconds before the first action
			time.Sleep(10 * time.Second)

			log.Println("--> Sending TARE command to scale...")
			if err := myScale.Tare(true); err != nil {
				log.Printf("Error taring scale: %v", err)
			}

			// Wait again
			time.Sleep(5 * time.Second)

			log.Println("--> Reading battery level...")
			batt, err := myScale.ReadBatteryChargePercent(nil)
			if err != nil {
				log.Printf("Error reading battery: %v", err)
			} else {
				log.Printf("--> Battery level is %d%%", batt)
			}
		}
	}()

	// --- Main application loop ---
	// This loop will block and process weight updates as they come in.
	// It will automatically exit when the 'weightUpdates' channel is closed.
	// The channel will be closed by the mock scale's implementation when its
	// context is canceled (which we do via the signal handler).
	for update := range weightUpdates {
		if update.Error != nil {
			log.Printf("Error received on update channel: %v", update.Error)
			continue
		}
		log.Printf("Weight: %.2f %s", update.Value, update.Unit)
	}

	// This line will only be reached after the channel is closed.
	log.Println("Weight update channel closed. Connection terminated.")
	log.Println("Application finished gracefully.")
}
