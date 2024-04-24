## Golang Repository Pattern Public Implementation Package

This repository contains a Go package for implementing a repository pattern to interact with a database using GORM.

## Introduction

This Go package provides a flexible and generic implementation of the repository pattern for database operations. It includes methods for CRUD operations as well as querying with pagination support.

## Installation

1. **Install the package:**

   ```bash
   go get github.com/sepehr-dh99/gorm-public-repository-pattern
   ```

2. **Set up the database:**

   Update the database connection details in the `main.go` file or where ever you want. Ensure that you have database accessible.

3. **Build and run the project:**

   bashCopy code to run your project (in this case main.go)

   `go run main.go`

## Usage

To use this package, import it into your Go project and initialize a repository instance. You can then use the provided methods to perform database operations. Here's an example of how to use the package:

goCopy code

```go
package publicRepository

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID       uint `gorm:"primaryKey"`
	Name     string
	Email    string
	Password string
	Status   int64
	// Other fields...
}

func main() {
	// Initialize database connection
	dsn := "host=your-postgresql-host port=5432 user=your-postgresql-user dbname=your-database password=your-password sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	migrateErr := db.AutoMigrate(
		&User{},
	)

	if migrateErr != nil {
		log.Fatal(err)
	}

	// Here User is your model struct
	userRepository := NewMainRepository[User](db)

	// Example of finding user by ID and some custom query
	userID := uint(1)
	user, err := userRepository.FindById(&userID, func(d *gorm.DB) *gorm.DB {
		return d.
			Where("status = ?", 1).
			Order("created_at DESC")
	})

	if err != nil {
		log.Fatalf("failed to find user: %v", err)
	}

	fmt.Println("User found:", user)

	// Example of finding all users
	users, err := userRepository.FindAll()
	if err != nil {
		log.Fatalf("failed to find users: %v", err)
	}

	fmt.Println("All users:", users)

	// Other CRUD operations and queries can be performed similarly
}
```
