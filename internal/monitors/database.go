package monitors

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/monocle-dev/monocle/internal/types"

	// Database drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func CheckDatabase(config *types.DatabaseConfig) error {
	timeout := config.Timeout

	if timeout == 0 {
		timeout = 10
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var dsn string

	switch config.Type {
	case "postgres", "postgresql":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			config.Username, config.Password, config.Host, config.Port, config.Database)
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	// Use correct driver names for sql.Open
	driverName := config.Type
	if config.Type == "postgresql" {
		driverName = "postgres"
	}

	db, err := sql.Open(driverName, dsn)

	if err != nil {
		return fmt.Errorf("failed to open a database connection: %v", err)
	}

	defer db.Close()

	// Test the connection with a ping
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	return nil
}
