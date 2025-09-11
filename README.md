# Repair Service Server

A comprehensive Golang backend server for the repair service application with enterprise-ready architecture.

## 🏗️ Architecture

```
server/
├── main.go              # Application entry point
├── go.mod               # Go module dependencies
├── config/              # Configuration management
│   └── config.go
├── database/            # Database connection and migrations
│   └── database.go
├── models/              # Database models
│   ├── user.go
│   ├── service.go
│   ├── booking.go
│   └── worker.go
├── middleware/          # HTTP middleware
│   ├── auth.go
│   └── logger.go
├── routes/              # API routes
│   ├── auth.go
│   └── routes.go
└── utils/               # Utility functions
    └── auth.go
```

## 🚀 Features

- **JWT Authentication** with phone number login
- **Role-based Access Control** (Customer, Worker, Admin)
- **PostgreSQL Database** with GORM ORM
- **RESTful API** with proper error handling
- **CORS Support** for cross-origin requests
- **Request Logging** and recovery middleware
- **Password Hashing** with bcrypt
- **Phone Number Validation** with +222 country code

## 📋 Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Git

## 🛠️ Setup Instructions

### 1. Clone the Repository

```bash
git clone <repository-url>
cd server
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Database Setup

Create a PostgreSQL database:

```sql
CREATE DATABASE repair_service_db;
```

### 4. Environment Configuration

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

# Phone Number Configuration
DEFAULT_COUNTRY_CODE=+222
```

### 5. Run the Server

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## 📚 API Documentation

### Authentication Endpoints

#### POST /api/v1/auth/signup

Register a new user account.

**Request Body:**

```json
{
  "phone_number": "+222123456789",
  "password": "password123",
  "full_name": "John Doe"
}
```

**Response:**

```json
{
  "message": "User registered successfully",
  "data": {
    "token": "jwt_token_here",
    "user": {
      "id": 1,
      "full_name": "John Doe",
      "phone_number": "+222123456789",
      "role": "customer",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

#### POST /api/v1/auth/signin

Authenticate existing user.

**Request Body:**

```json
{
  "phone_number": "+222123456789",
  "password": "password123"
}
```

**Response:**

```json
{
  "message": "Authentication successful",
  "data": {
    "token": "jwt_token_here",
    "user": {
      "id": 1,
      "full_name": "John Doe",
      "phone_number": "+222123456789",
      "role": "customer",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

### Protected Endpoints

All protected endpoints require the `Authorization` header:

```
Authorization: Bearer <jwt_token>
```

#### GET /api/v1/users/profile

Get current user profile.

#### PUT /api/v1/users/profile

Update current user profile.

#### GET /api/v1/services

Get all available services.

#### GET /api/v1/services/:id

Get specific service details.

#### GET /api/v1/services/categories

Get service categories.

#### POST /api/v1/bookings

Create a new booking.

#### GET /api/v1/bookings

Get user's bookings.

#### GET /api/v1/bookings/:id

Get specific booking details.

#### PUT /api/v1/bookings/:id/cancel

Cancel a booking.

#### GET /api/v1/workers

Get all workers.

#### GET /api/v1/workers/:id

Get specific worker details.

#### GET /api/v1/workers/available

Get available workers.

## 🔐 Authentication Flow

1. **Registration**: User provides phone number, password, and full name
2. **Phone Validation**: Server validates +222 format and uniqueness
3. **Password Hashing**: Password is hashed using bcrypt
4. **User Creation**: User is created with customer role
5. **Token Generation**: JWT token is generated and returned
6. **Login**: User provides phone number and password
7. **Authentication**: Server validates credentials and returns token
8. **Protected Access**: Token is used for subsequent API calls

## 🗄️ Database Schema

### Users Table

```sql
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    full_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('customer', 'worker', 'admin') NOT NULL,
    profile_picture_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE
);
```

### Services Table

```sql
CREATE TABLE services (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    duration VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL,
    image_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### Bookings Table

```sql
CREATE TABLE bookings (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    service_id INT NOT NULL,
    worker_id INT,
    status ENUM('pending', 'accepted', 'in_progress', 'completed', 'cancelled') DEFAULT 'pending',
    address VARCHAR(500) NOT NULL,
    date DATETIME NOT NULL,
    time VARCHAR(20) NOT NULL,
    notes TEXT,
    total_price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (service_id) REFERENCES services(id),
    FOREIGN KEY (worker_id) REFERENCES workers(id)
);
```

### Workers Table

```sql
CREATE TABLE workers (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT UNIQUE NOT NULL,
    specialization VARCHAR(255) NOT NULL,
    experience VARCHAR(100) NOT NULL,
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_jobs INT DEFAULT 0,
    is_available BOOLEAN DEFAULT TRUE,
    current_location VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## 🔧 Development

### Running in Development Mode

```bash
GIN_MODE=debug go run main.go
```

### Running Tests

```bash
go test ./...
```

### Building for Production

```bash
go build -o repair-service-server main.go
```

## 🚀 Deployment

### Docker (Recommended)

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o repair-service-server main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/repair-service-server .
COPY --from=builder /app/.env .

EXPOSE 8080
CMD ["./repair-service-server"]
```

Build and run:

```bash
docker build -t repair-service-server .
docker run -p 8080:8080 repair-service-server
```

## 🔒 Security Considerations

- JWT tokens expire after 24 hours (configurable)
- Passwords are hashed using bcrypt
- Phone numbers are validated for +222 format
- CORS is configured for security
- All sensitive data is excluded from JSON responses
- Input validation on all endpoints

## 📝 Environment Variables

| Variable               | Description                | Default                     |
| ---------------------- | -------------------------- | --------------------------- |
| `PORT`                 | Server port                | `8080`                      |
| `GIN_MODE`             | Gin framework mode         | `debug`                     |
| `DB_HOST`              | Database host              | `localhost`                 |
| `DB_PORT`              | Database port              | `3306`                      |
| `DB_USER`              | Database username          | `root`                      |
| `DB_PASSWORD`          | Database password          | `password`                  |
| `DB_NAME`              | Database name              | `repair_service_db`         |
| `JWT_SECRET`           | JWT signing secret         | `your-super-secret-jwt-key` |
| `JWT_EXPIRY_HOURS`     | JWT token expiry hours     | `24`                        |
| `DEFAULT_COUNTRY_CODE` | Default phone country code | `+222`                      |

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.
"# rft2649" 
