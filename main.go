package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

const (
	flightID      = "SG-101"
	targetSeat    = "1A"
	numPassengers = 10
	numSeats      = 10
)

func main() {
	// Build DSN from environment variables
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Could not open DB connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Could not ping DB: %v", err)
	}
	log.Println("Successfully connected to the database!")

	targetSeatID, err := SetupDatabase(db)
	if err != nil {
		log.Fatalf("Failed to set up database: %v", err)
	}
	defer CleanupDatabase(db)

	// Reset for the next run
	if err := ResetData(db, targetSeatID); err != nil {
		log.Fatalf("Could not reset data: %v", err)
	}

	// --- Simulation: WITH Pessimistic Locking ---
	var wg sync.WaitGroup
	fmt.Println("\n==================================================")
	log.Println("🚀 Starting Simulation WITH Pessimistic Locking")
	fmt.Println("==================================================")
	for i := 1; i <= numPassengers; i++ {
		wg.Add(1)
		go func(passengerNum int) {
			defer wg.Done()
			BookSeat(db, strconv.Itoa(passengerNum), targetSeatID, LockModeWait)
		}(i)
	}
	wg.Wait()
	ResetData(db, targetSeatID)

	fmt.Println("\n==================================================")
	log.Println("🚀 Starting Simulation with SKIP LOCKED (Booking ANY Seat)")
	fmt.Println("==================================================")
	var wgSkip sync.WaitGroup
	for i := 1; i <= numPassengers; i++ {
		wgSkip.Add(1)
		go func(passengerNum int) {
			defer wgSkip.Done()
			BookAnyAvailableSeat(db, strconv.Itoa(passengerNum))
		}(i)
	}
	wgSkip.Wait()
	log.Println("🏁 SKIP LOCKED simulation finished.")

	log.Println("🏁 Simulation WITH lock finished.")
}
