package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/go-sql-driver/mysql"
)

type LockMode int

const (
	LockModeWait LockMode = iota // Default FOR UPDATE
	LockModeNoWait
)

// SetupDatabase connects to the DB, drops the old table, creates a new one, and seeds it.
func SetupDatabase(db *sql.DB) (int, error) {
	log.Println("--- Database Setup ---")
	// Drop table if it exists for a clean slate
	if _, err := db.Exec("DROP TABLE IF EXISTS seats"); err != nil {
		return 0, fmt.Errorf("failed to drop table: %w", err)
	}
	log.Println("Dropped existing 'seats' table.")

	// Create the seats table
	createTableSQL := `
	CREATE TABLE seats (
		id INT AUTO_INCREMENT PRIMARY KEY,
		flight_id VARCHAR(255) NOT NULL,
		seat_number VARCHAR(10) NOT NULL,
		status ENUM('available', 'occupied') DEFAULT 'available',
		passenger_id VARCHAR(255),
		UNIQUE KEY (flight_id, seat_number)
	);`
	if _, err := db.Exec(createTableSQL); err != nil {
		return 0, fmt.Errorf("failed to create table: %w", err)
	}
	log.Println("Created new 'seats' table.")

	// Seed the table with available seats
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	stmt, err := tx.Prepare("INSERT INTO seats(flight_id, seat_number) VALUES(?, ?)")
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for i := 0; i < numSeats; i++ {
		seatNum := fmt.Sprintf("%d%c", (i/8)+1, 'A'+(i%8))
		if _, err := stmt.Exec(flightID, seatNum); err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to insert seat %s: %w", seatNum, err)
		}
	}
	log.Printf("Inserted %d seats for flight %s.\n", numSeats, flightID)

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get the ID of our target seat
	var targetSeatID int
	err = db.QueryRow("SELECT id FROM seats WHERE flight_id = ? AND seat_number = ?", flightID, targetSeat).Scan(&targetSeatID)
	if err != nil {
		return 0, fmt.Errorf("failed to get ID for target seat %s: %w", targetSeat, err)
	}
	log.Printf("Target seat %s has ID: %d\n", targetSeat, targetSeatID)
	log.Println("--- Setup Complete ---")
	return targetSeatID, nil
}

// ResetData sets the target seat back to 'available' for the second simulation run.
func ResetData(db *sql.DB, seatID int) error {
	log.Println("\n--- Resetting Data for Next Simulation ---")
	_, err := db.Exec("UPDATE seats SET status = 'available', passenger_id = NULL WHERE id = ?", seatID)
	if err != nil {
		return fmt.Errorf("failed to reset data: %w", err)
	}
	log.Printf("Seat %d has been reset to 'available'.\n", seatID)
	return nil
}

// CleanupDatabase drops the table.
func CleanupDatabase(db *sql.DB) {
	log.Println("\n--- Cleaning up ---")
	if _, err := db.Exec("DROP TABLE seats"); err != nil {
		log.Printf("Failed to drop table during cleanup: %v", err)
	} else {
		log.Println("Dropped 'seats' table.")
	}
}

// BookSeat attempts to book a specific seat for a passenger.
// The useLocking flag determines whether to use SELECT ... FOR UPDATE.
func BookSeat(db *sql.DB, passengerID string, seatID int, mode LockMode) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[Passenger-%s] ❌ Failed to start transaction: %v", passengerID, err)
		return
	}
	defer tx.Rollback()

	var currentStatus string
	query := "SELECT status FROM seats WHERE id = ? FOR UPDATE"
	if mode == LockModeNoWait {
		query += " NOWAIT"
	}

	err = tx.QueryRow(query, seatID).Scan(&currentStatus)
	if err != nil {
		// With NOWAIT, the DB returns a specific error if the row is locked.
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 3572 {
			log.Printf("[Passenger-%s] 😞 Seat %d is locked by another user. Not waiting.", passengerID, seatID)
		} else {
			log.Printf("[Passenger-%s] ❌ Failed to query seat status: %v", passengerID, err)
		}
		return
	}

	log.Printf("[Passenger-%s] 👀 Sees seat %d is '%s'.", passengerID, seatID, currentStatus)

	time.Sleep(100 * time.Millisecond)

	if currentStatus == "available" {
		log.Printf("[Passenger-%s] ✅ Seat is available. Attempting to book...", passengerID)
		_, err := tx.Exec("UPDATE seats SET status = 'occupied', passenger_id = ? WHERE id = ?", "Passenger-"+passengerID, seatID)
		if err != nil {
			log.Printf("[Passenger-%s] ❌ Failed to update seat: %v", passengerID, err)
			return
		}
		log.Printf("[Passenger-%s] 🎉 Successfully booked seat %d!", passengerID, seatID)
	} else {
		log.Printf("[Passenger-%s] 😞 Failed to book seat %d, it was already '%s'.", passengerID, seatID, currentStatus)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[Passenger-%s] ❌ Failed to commit transaction: %v", passengerID, err)
	}
}

// BookAnyAvailableSeat finds the first available seat, skips any that are locked, and books it.
func BookAnyAvailableSeat(db *sql.DB, passengerID string) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[Passenger-%s] ❌ Failed to start transaction: %v", passengerID, err)
		return
	}
	defer tx.Rollback()

	var seatID int
	var seatNumber string

	// Find an available seat, but skip any that other workers are currently looking at.
	query := `
		SELECT id, seat_number FROM seats 
		WHERE status = 'available' 
		LIMIT 1 
		FOR UPDATE SKIP LOCKED`

	err = tx.QueryRow(query).Scan(&seatID, &seatNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[Passenger-%s] 😞 No available seats found.", passengerID)
		} else {
			log.Printf("[Passenger-%s] ❌ Error finding a seat: %v", passengerID, err)
		}
		return
	}

	// We have a lock on this specific seat (seatID). Now book it.
	log.Printf("[Passenger-%s] ✅ Found and locked available seat %s (ID: %d). Booking now...", passengerID, seatNumber, seatID)
	_, err = tx.Exec("UPDATE seats SET status = 'occupied', passenger_id = ? WHERE id = ?", "Passenger-"+passengerID, seatID)
	if err != nil {
		return
	}

	if err := tx.Commit(); err == nil {
		log.Printf("[Passenger-%s] 🎉 Successfully booked seat %s!", passengerID, seatNumber)
	}
}
