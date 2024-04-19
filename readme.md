## Go Repository Pattern Package

This repository contains a Go package for implementing a repository pattern to interact with a PostgreSQL database using GORM.

## Introduction

This Go package provides a flexible and generic implementation of the repository pattern for database operations. It includes methods for CRUD operations as well as querying with pagination support.

## Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/sepehr-dh99/gorm-public-repository-pattern
   ```

2. **Install dependencies:**

   This project requires Go, [GORM](https://gorm.io/) and Gorm Postgres driver to be installed. You can install GORM and Gorm Postgres driver using the following command:

   bashCopy code

   `go get -u gorm.io/gorm`
   `go get gorm.io/driver/postgres`

3. **Set up the database:**

   Update the database connection details in the `main.go` file. Ensure that you have a PostgreSQL database accessible.

4. **Build and run the project:**

   bashCopy code

   `go run main.go`

## Usage

To use this package, import it into your Go project and initialize a repository instance. You can then use the provided methods to perform database operations. Here's an example of how to use the package:

goCopy code

```go
package main

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sepehr-dh99/gorm-public-repository-pattern/repository"
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
	repository.DB = db

	// Usage example
	userRepository := repository.NewMainRepository[User]()
	// Here User is your model struct

	// Example of finding user by ID
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
