# Pessimistic Locking Demo in Go & MySQL

This project is a simple command-line application designed to demonstrate database locking concepts using a simulated airline check-in system.

It illustrates the various pessimistic locking strategies:
* **`FOR UPDATE` (Wait):** The default pessimistic lock where other transactions wait.
* **`FOR UPDATE NOWAIT`:** A non-blocking lock that fails immediately if a resource is busy.
* **`FOR UPDATE SKIP LOCKED`:** A strategy for parallel processing where locked resources are ignored.

## Prerequisites

* [Go](https://go.dev/doc/install) (version 1.18 or newer)
* [Docker](https://www.docker.com/get-started/)

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

The Go application needs to know how to connect to the database. You can do this in your terminal or by [configuring the `launch.json` file in VS Code](https://code.visualstudio.com/docs/editor/debugging#_launch-configurations).

```bash
# For Linux / macOS
export DB_USER=root
export DB_PASS=my-secret-pw
export DB_HOST=127.0.0.1
export DB_PORT=3306
export DB_NAME=locking_demo
```

**B. Initialize Go Module and Run**

Place all `.go` files in the same directory. Then navigate to that directory in your terminal and run these commands:

```bash
# Initialize a go module (only needs to be done once)
go mod init pessimistic-locking-demo
# Tidy will fetch the mysql driver dependency
go mod tidy
# Run the application! This will run all simulations in sequence.
go run .
```

## Expected Output

The application will run 3 distinct simulations.

### Simulation 1: WITH `FOR UPDATE` (The "Wait" Solution 👍)
Here, 10 passengers again try to book the **same seat**. The first one gets a lock, and all others are forced to wait in a queue.

```
...
[Passenger-3] 🎉 Successfully booked seat 1!
(The other 9 passengers wait here, blocked by the database lock...)
[Passenger-1] 😞 Failed to book seat 1, it was already 'occupied'.
[Passenger-7] 😞 Failed to book seat 1, it was already 'occupied'.
...
```
**Result:** Correctness. Exactly one passenger succeeds, and the others are correctly told the seat is taken.

---
### Simulation 2: WITH `FOR UPDATE NOWAIT` (The "Fail Fast" Solution 😠)
This simulation is the same as above, but instead of waiting, transactions will fail instantly if the seat is already locked.

```
...
[Passenger-6] 🎉 Successfully booked seat 1!
[Passenger-2] 😞 Seat 1 is locked by another user. Not waiting.
[Passenger-8] 😞 Seat 1 is locked by another user. Not waiting.
[Passenger-1] 😞 Seat 1 is locked by another user. Not waiting.
...
```
**Result:** Immediate feedback. One passenger succeeds, and the other nine fail instantly without freezing, which is ideal for interactive user interfaces.

---
### Simulation 3: WITH `FOR UPDATE SKIP LOCKED` (The "Parallel Worker" Solution 🧑‍🏭)
This simulation demonstrates a different scenario: 10 passengers try to book **any available seat** concurrently. `SKIP LOCKED` allows each transaction to ignore seats locked by others and grab the next free one.

```
...
[Passenger-10] ✅ Found and locked available seat 1J (ID: 10). Booking now...
[Passenger-4] ✅ Found and locked available seat 1D (ID: 4). Booking now...
[Passenger-1] ✅ Found and locked available seat 1A (ID: 1). Booking now...
[Passenger-10] 🎉 Successfully booked seat 1J!
[Passenger-4] 🎉 Successfully booked seat 1D!
[Passenger-1] 🎉 Successfully booked seat 1A!
...
```
**Result:** High-performance parallel processing. All 10 passengers succeed almost simultaneously by booking 10 *different* seats. This is the ideal pattern for processing job queues.