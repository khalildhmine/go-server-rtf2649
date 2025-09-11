package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"repair-service-server/models"
)

var DB *gorm.DB

// Initialize sets up the database connection and runs migrations
func Initialize() error {
	// Production: require full Postgres URL from DB_URL
	// Example: DB_URL=postgresql://user:pass@host:port/dbname?sslmode=require
	connString := os.Getenv("DB_URL")
	if connString == "" {
		return fmt.Errorf("DB_URL is required in production. Set DB_URL to a valid Postgres URL")
	}

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Open database connection
	var err error
	DB, err = gorm.Open(postgres.Open(connString), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying SQL database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Successfully connected to database")

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Database migrations completed successfully")

	return nil
}

// runMigrations creates or updates database tables
func runMigrations() error {
	// First, migrate tables that don't have data migration issues
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.ServiceCategory{},
		&models.Service{},
		&models.WorkerProfile{},
		&models.CustomerServiceRequest{},
		&models.WorkerResponse{},
	); err != nil {
		return err
	}

	// Handle services table migration manually to avoid NOT NULL constraint issues
	if err := migrateServicesTable(); err != nil {
		return err
	}

	// Handle worker_profiles table migration manually to avoid NOT NULL constraint issues
	if err := migrateWorkerProfilesTable(); err != nil {
		return err
	}

	// Handle addresses table migration manually to ensure foreign key constraints
	if err := migrateAddressesTable(); err != nil {
		return err
	}

	// Ensure service_categories.icon has varchar(255)
	if err := migrateServiceCategoriesIconLength(); err != nil {
		return err
	}

	return nil
}

// migrateServicesTable handles the services table migration manually
func migrateServicesTable() error {
	// Check if services table exists
	if !DB.Migrator().HasTable(&models.Service{}) {
		// Create new table
		return DB.AutoMigrate(&models.Service{})
	}

	// Check if category_id column exists
	if !DB.Migrator().HasColumn(&models.Service{}, "category_id") {
		// First, check if there's existing data
		var count int64
		DB.Model(&models.Service{}).Count(&count)
		
		if count > 0 {
			// There's existing data, we need to handle this carefully
			// Add category_id as nullable first
			if err := DB.Exec("ALTER TABLE services ADD COLUMN category_id bigint").Error; err != nil {
				return err
			}
			
			// Update existing records with a default category_id (assuming category 1 exists)
			if err := DB.Exec("UPDATE services SET category_id = 1 WHERE category_id IS NULL").Error; err != nil {
				return err
			}
			
			// Now make it NOT NULL
			if err := DB.Exec("ALTER TABLE services ALTER COLUMN category_id SET NOT NULL").Error; err != nil {
				return err
			}
			
			log.Println("✅ Successfully migrated services table with category_id")
		} else {
			// No existing data, safe to add NOT NULL constraint
			if err := DB.Exec("ALTER TABLE services ADD COLUMN category_id bigint NOT NULL").Error; err != nil {
				return err
			}
		}
	}

	// Check if old category column still exists and remove it
	if DB.Migrator().HasColumn(&models.Service{}, "category") {
		// Drop the old category column
		if err := DB.Exec("ALTER TABLE services DROP COLUMN category").Error; err != nil {
			log.Printf("⚠️  Could not drop old category column: %v", err)
		} else {
			log.Println("✅ Successfully dropped old category column")
		}
	}

	return nil
}

// migrateWorkerProfilesTable handles the worker_profiles table migration manually
func migrateWorkerProfilesTable() error {
	// Check if worker_profiles table exists
	if !DB.Migrator().HasTable(&models.WorkerProfile{}) {
		// Create new table
		return DB.AutoMigrate(&models.WorkerProfile{})
	}

	// Check if category_id column exists
	if !DB.Migrator().HasColumn(&models.WorkerProfile{}, "category_id") {
		// First, check if there's existing data
		var count int64
		DB.Model(&models.WorkerProfile{}).Count(&count)
		
		if count > 0 {
			// There's existing data, we need to handle this carefully
			// Add category_id as nullable first
			if err := DB.Exec("ALTER TABLE worker_profiles ADD COLUMN category_id bigint").Error; err != nil {
				return err
			}
			
			// Update existing records with a default category_id (assuming category 1 exists)
			if err := DB.Exec("UPDATE worker_profiles SET category_id = 1 WHERE category_id IS NULL").Error; err != nil {
				return err
			}
			
			// Now make it NOT NULL
			if err := DB.Exec("ALTER TABLE worker_profiles ALTER COLUMN category_id SET NOT NULL").Error; err != nil {
				return err
			}
			
			log.Println("✅ Successfully migrated worker_profiles table with category_id")
		} else {
			// No existing data, safe to add NOT NULL constraint
			if err := DB.Exec("ALTER TABLE worker_profiles ADD COLUMN category_id bigint NOT NULL").Error; err != nil {
				return err
			}
		}
	}

	// Check if old category column still exists and remove it
	if DB.Migrator().HasColumn(&models.WorkerProfile{}, "category") {
		// Drop the old category column
		if err := DB.Exec("ALTER TABLE worker_profiles DROP COLUMN category").Error; err != nil {
			log.Printf("⚠️  Could not drop old category column: %v", err)
		} else {
			log.Println("✅ Successfully dropped old category column")
		}
	}

	return nil
}

// migrateAddressesTable handles the addresses table migration manually
func migrateAddressesTable() error {
	// Check if addresses table exists
	if !DB.Migrator().HasTable(&models.Address{}) {
		// Create new table
		return DB.AutoMigrate(&models.Address{})
	}

	// Check if user_id column exists
	if !DB.Migrator().HasColumn(&models.Address{}, "user_id") {
		// First, check if there's existing data
		var count int64
		DB.Model(&models.Address{}).Count(&count)
		
		if count > 0 {
			// There's existing data, we need to handle this carefully
			// Add user_id as nullable first
			if err := DB.Exec("ALTER TABLE addresses ADD COLUMN user_id bigint").Error; err != nil {
				return err
			}
			
			// Update existing records with a default user_id (assuming user 1 exists)
			if err := DB.Exec("UPDATE addresses SET user_id = 1 WHERE user_id IS NULL").Error; err != nil {
				return err
			}
			
			// Now make it NOT NULL
			if err := DB.Exec("ALTER TABLE addresses ALTER COLUMN user_id SET NOT NULL").Error; err != nil {
				return err
			}
			
			log.Println("✅ Successfully migrated addresses table with user_id")
		} else {
			// No existing data, safe to add NOT NULL constraint
			if err := DB.Exec("ALTER TABLE addresses ADD COLUMN user_id bigint NOT NULL").Error; err != nil {
				return err
			}
		}
	}

	// Check if old user column still exists and remove it
	if DB.Migrator().HasColumn(&models.Address{}, "user") {
		// Drop the old user column
		if err := DB.Exec("ALTER TABLE addresses DROP COLUMN user").Error; err != nil {
			log.Printf("⚠️  Could not drop old user column: %v", err)
		} else {
			log.Println("✅ Successfully dropped old user column")
		}
	}

	return nil
}

// migrateServiceCategoriesIconLength ensures icon column is varchar(255)
func migrateServiceCategoriesIconLength() error {
    // Only run if table exists
    if !DB.Migrator().HasTable(&models.ServiceCategory{}) {
        return nil
    }

    // Try altering the column type to varchar(255)
    if err := DB.Exec("ALTER TABLE service_categories ALTER COLUMN icon TYPE varchar(255)").Error; err != nil {
        return err
    }
    return nil
}

func GetDB() *gorm.DB {
	return DB
}
