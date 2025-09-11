# PostgreSQL Setup Guide

## 🐘 Installing PostgreSQL

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

### macOS (using Homebrew)

```bash
brew install postgresql
brew services start postgresql
```

### Windows

Download and install from: https://www.postgresql.org/download/windows/

## 🔧 Initial Setup

### 1. Start PostgreSQL Service

```bash
# Ubuntu/Debian
sudo systemctl start postgresql
sudo systemctl enable postgresql

# macOS
brew services start postgresql
```

### 2. Access PostgreSQL

```bash
sudo -u postgres psql
```

### 3. Create Database and User

```sql
-- Create database
CREATE DATABASE repair_service_db;

-- Create user (optional, you can use postgres user)
CREATE USER repair_user WITH PASSWORD 'your_password_here';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE repair_service_db TO repair_user;

-- Exit
\q
```

## 🔐 Configure Environment

Create a `.env` file in the server directory:

```env
# Server Configuration
PORT=8080
GIN_MODE=debug

# PostgreSQL Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_NAME=repair_service_db
DB_SSL_MODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
JWT_EXPIRY_HOURS=24

# Phone Configuration
DEFAULT_COUNTRY_CODE=+222

# Expo Push Notifications
EXPO_ACCESS_TOKEN=your_expo_access_token_here
```

## 🚀 Test Connection

### 1. Test with psql

```bash
psql -h localhost -p 5432 -U postgres -d repair_service_db
```

### 2. Test with Go Server

```bash
go run main.go
```

## 🔍 Troubleshooting

### Connection Issues

- Check if PostgreSQL is running: `sudo systemctl status postgresql`
- Verify port: `netstat -tlnp | grep 5432`
- Check firewall settings

### Authentication Issues

- Edit `pg_hba.conf` for authentication method
- Reset postgres password: `sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'new_password';"`

### SSL Issues

- Set `DB_SSL_MODE=disable` for local development
- For production, configure proper SSL certificates

## 📊 Database Management

### Useful Commands

```sql
-- List databases
\l

-- Connect to database
\c repair_service_db

-- List tables
\dt

-- Describe table
\d table_name

-- Exit
\q
```

### Backup and Restore

```bash
# Backup
pg_dump -h localhost -U postgres repair_service_db > backup.sql

# Restore
psql -h localhost -U postgres repair_service_db < backup.sql
```
