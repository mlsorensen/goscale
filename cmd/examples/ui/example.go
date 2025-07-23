package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mlsorensen/goscale"
	// This tells the Go compiler to include the package, which runs its init()
	// function. The init() function, in turn, calls goscale.Register(). You can
	// specify specific scales individually or just "all"
	_ "github.com/mlsorensen/goscale/pkg/scales/all"
)

func main() {
	a := app.New()
	w := a.NewWindow("Scale App")
	dev, err := goscale.ScanForOne(10 * time.Second)

	if err != nil {
		log.Fatal(err)
	}

	myScale, err := goscale.NewScaleForDevice(dev)
	if err != nil {
		log.Fatalf("Fatal: Could not create scale instance: %v", err)
	}

	displayNameLabel := widget.NewLabel(myScale.DisplayName())
	weightLabel := widget.NewLabel("")
	batteryLabel := widget.NewLabel("")
	sleepTimeoutLabel := widget.NewLabel("")
	var wg sync.WaitGroup
	tareButton := widget.NewButton("Tare", func() {
		log.Println("-------------------------> Sending TARE command to scale...")
		if err := myScale.Tare(true); err != nil {
			log.Printf("Error taring scale: %v", err)
		}
	})

	adjSleepButton := widget.NewButton("Adjust Sleep Timer", func() {
		log.Println("advancing sleep timer")
		if err := myScale.AdvanceSleepTimeout(); err != nil {
			log.Printf("Error advancing sleep timer: %v", err)
		}
	})

	var shutdown chan os.Signal
	shutdown = make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	wg.Add(2)

	go func() {
		defer wg.Done()
		for sig := range shutdown {
			log.Println("Shutdown signal received:", sig)
			a.Quit()
		}
	}()

	features := myScale.GetFeatures()

	var weightUpdates <-chan goscale.WeightUpdate
	go func() {
		defer wg.Done()
		var err error
		weightUpdates, err = myScale.Connect()
		if err != nil {
			log.Fatalf("Fatal: Could not connect to scale: %v", err)
		}
		for update := range weightUpdates {
			if update.Error != nil {
				log.Printf("Error received on update channel: %v", update.Error)
				continue
			}
			fyne.Do(func() {
				weightLabel.SetText(fmt.Sprintf("weight: %.2f %s", update.Value, update.Unit))
				if features.BatteryPercent {
					battPct, _ := myScale.GetBatteryChargePercent()
					batteryLabel.SetText(fmt.Sprintf("battery: %.1f%%", battPct))
				}
				if features.SleepTimeout {
					sleepTimeoutLabel.SetText(fmt.Sprintf("sleep timeout: %s", myScale.GetSleepTimeout()))
				}
			})
		}
		if err := myScale.Disconnect(); err != nil {
			log.Printf("Error disconnecting from scale: %v", err)
		}
	}()

	ctr := container.NewVBox()
	ctr.Add(displayNameLabel)
	ctr.Add(weightLabel)

	if features.BatteryPercent {
		ctr.Add(batteryLabel)
	}

	if features.SleepTimeout {
		ctr.Add(sleepTimeoutLabel)
	}

	if features.Tare {
		ctr.Add(tareButton)
	}

	if features.SleepTimeout {
		ctr.Add(adjSleepButton)
	}

	w.SetContent(ctr)

	go func() {
		wg.Wait()
		if err := myScale.Disconnect(); err != nil {
			log.Printf("Error disconnecting from scale: %v", err)
		}
	}()

	w.ShowAndRun()
}
