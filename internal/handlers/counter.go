package handlers

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type counterData struct {
	Scansioni int64 `json:"scansioni"`
}

var (
	counterMu       sync.Mutex
	counterValue    int64
	pendingWrites   int
	counterFilePath = "counter.json"
	flushTicker     *time.Ticker
)

const flushEveryN = 10
const flushInterval = 30 * time.Second

func InitCounter() {
	counterMu.Lock()
	defer counterMu.Unlock()

	data, err := os.ReadFile(counterFilePath)
	if err != nil {
		counterValue = 14327
		log.Printf("[counter] counter.json non trovato, seed iniziale: %d", counterValue)
	} else {
		var cd counterData
		if err := json.Unmarshal(data, &cd); err != nil {
			counterValue = 14327
			log.Printf("[counter] Errore parsing counter.json, seed iniziale: %d", counterValue)
		} else {
			counterValue = cd.Scansioni
			log.Printf("[counter] Caricato contatore: %d", counterValue)
		}
	}

	flushTicker = time.NewTicker(flushInterval)
	go func() {
		for range flushTicker.C {
			flushCounter()
		}
	}()
}

func IncrementCounter() int64 {
	counterMu.Lock()
	counterValue++
	val := counterValue
	pendingWrites++
	shouldFlush := pendingWrites >= flushEveryN
	counterMu.Unlock()

	if shouldFlush {
		flushCounter()
	}
	return val
}

func GetCounter() int64 {
	counterMu.Lock()
	defer counterMu.Unlock()
	return counterValue
}

func flushCounter() {
	counterMu.Lock()
	if pendingWrites == 0 {
		counterMu.Unlock()
		return
	}
	val := counterValue
	pendingWrites = 0
	counterMu.Unlock()

	cd := counterData{Scansioni: val}
	data, err := json.Marshal(cd)
	if err != nil {
		log.Printf("[counter] Errore marshaling: %v", err)
		return
	}
	if err := os.WriteFile(counterFilePath, data, 0644); err != nil {
		log.Printf("[counter] Errore scrittura counter.json: %v", err)
	}
}
