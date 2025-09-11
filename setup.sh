#!/bin/bash

echo "🚀 Setting up Repair Service Server..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

echo "✅ Go is installed"

# Check if PostgreSQL is running
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL is not installed. Please install PostgreSQL 12 or higher."
    exit 1
fi

echo "✅ PostgreSQL is installed"

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📝 Creating .env file..."
    cat > .env << EOF
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

# Phone Number Configuration
DEFAULT_COUNTRY_CODE=+222
EOF
    echo "✅ .env file created"
else
    echo "✅ .env file already exists"
fi

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

if [ $? -eq 0 ]; then
    echo "✅ Dependencies installed successfully"
else
    echo "❌ Failed to install dependencies"
    exit 1
fi

# Test database connection
echo "🔍 Testing database connection..."
go run main.go &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test health endpoint
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Server is running and healthy"
else
    echo "❌ Server health check failed"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

# Stop the server
kill $SERVER_PID 2>/dev/null

echo ""
echo "🎉 Setup completed successfully!"
echo ""
echo "To start the server:"
echo "  go run main.go"
echo ""
echo "To test the API:"
echo "  curl http://localhost:8080/health"
echo ""
echo "📚 Check README.md for API documentation"