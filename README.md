# Pessimistic Locking Demo in Go & MySQL

This project is a simple command-line application designed to demonstrate the concept of **pessimistic locking** in a database context.

We simulate a common race condition: an airline check-in system where multiple passengers try to book the exact same seat at the same time.

-   **Run 1 (Without Lock):** Shows the race condition where multiple passengers believe they have successfully booked the same seat.
-   **Run 2 (With Lock):** Shows how pessimistic locking (`SELECT ... FOR UPDATE`) correctly prevents this, ensuring only one passenger can book the seat.

## Prerequisites

-   [Go](https://go.dev/doc/install) (version 1.18 or newer)
-   [Docker](https://www.docker.com/get-started/)

## 1. Setup the Database

First, we need a MySQL database. The easiest way is to run it in a Docker container.

Open your terminal and run the following command. This will start a MySQL 8 container, create a database named `locking_demo`, and set the root password.

```bash
docker run --name mysql-pessimistic-lock-demo -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=my-secret-pw \
  -e MYSQL_DATABASE=locking_demo \
  -d mysql:8
```
*It may take a minute for the database server to initialize inside the container.*

## 2. Run the Go Application

Follow these steps to run the Go program.

**A. Set Environment Variables**

The Go application needs to know how to connect to the database you just started.

```bash
# For Linux / macOS
export DB_USER=root
export DB_PASS=my-secret-pw
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_NAME=locking_demo

# For Windows (Command Prompt)
# set DB_USER=root
# set DB_PASS=my-secret-pw
# ... and so on
```

**B. Initialize Go Module and Run**

Navigate to the directory containing `main.go` and run these commands:

```bash
# Initialize a go module (only needs to be done once)
go mod init pessimistic-locking-demo
# Tidy will fetch the mysql driver dependency
go mod tidy
# Run the application!
go run main.go
```

## Expected Output

You will see the output of two simulations.

#### Simulation 1: WITHOUT Locking (The Problem 👎)

In the first run, you'll see that multiple passengers read the status as 'available' before anyone can update it. This leads to several "successful" bookings for the same seat, which is a critical data integrity error.

```
...
🚀 Starting Simulation WITHOUT Pessimistic Locking
...
[Passenger-5] 👀 Sees seat 1 is 'available'.
[Passenger-2] 👀 Sees seat 1 is 'available'.
[Passenger-8] 👀 Sees seat 1 is 'available'.
...
[Passenger-5] ✅ Seat is available. Attempting to book...
[Passenger-2] ✅ Seat is available. Attempting to book...
[Passenger-8] ✅ Seat is available. Attempting to book...
...
[Passenger-5] 🎉 Successfully booked seat 1!
[Passenger-2] 🎉 Successfully booked seat 1!
[Passenger-8] 🎉 Successfully booked seat 1!
...
```
**Result:** Chaos! Multiple passengers think they have seat `1A`. The last one to write to the database is the "winner," but the others have already received incorrect confirmation.

#### Simulation 2: WITH Pessimistic Locking (The Solution 👍)

In the second run, `SELECT ... FOR UPDATE` is used. The first passenger to execute this query places a lock on the row. All other passengers are forced to wait until that first transaction is committed. By then, the seat is already taken.

```
...
🚀 Starting Simulation WITH Pessimistic Locking
...
[Passenger-3] 👀 Sees seat 1 is 'available'.
[Passenger-3] ✅ Seat is available. Attempting to book...
[Passenger-3] 🎉 Successfully booked seat 1!
(The other 9 passengers wait here, blocked by the database lock...)
...
[Passenger-1] 👀 Sees seat 1 is 'occupied'.
[Passenger-1] 😞 Failed to book seat 1, it was already 'occupied'.
[Passenger-7] 👀 Sees seat 1 is 'occupied'.
[Passenger-7] 😞 Failed to book seat 1, it was already 'occupied'.
...
```
**Result:** Correctness! Exactly one passenger succeeds. The others are correctly informed that the seat is no longer available, preventing any double-booking.