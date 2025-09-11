# Notification System Documentation

## Overview

The notification system provides comprehensive push notification capabilities using Expo's push service, with detailed device tracking and user preference management.

## Features

- **Expo Push Notifications**: Integration with Expo's push service
- **Device Management**: Track multiple devices per user
- **Detailed Device Info**: Store comprehensive device information
- **User Preferences**: Granular notification preferences
- **Notification History**: Store and manage notification history
- **Quiet Hours**: Respect user's quiet hours settings
- **Retry Logic**: Automatic retry for failed notifications
- **Analytics**: Track notification delivery and engagement

## Database Schema

### Notifications Table

```sql
CREATE TABLE notifications (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    body VARCHAR(1000) NOT NULL,
    type ENUM('booking_created', 'booking_accepted', 'booking_in_progress', 'booking_completed', 'booking_cancelled', 'worker_assigned', 'payment_received', 'promotion', 'system') NOT NULL,
    status ENUM('pending', 'sent', 'delivered', 'failed', 'read') DEFAULT 'pending',
    data JSON,
    expo_push_token VARCHAR(255),
    device_id VARCHAR(255),
    device_type VARCHAR(50),
    device_brand VARCHAR(100),
    device_model VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    screen_width INT,
    screen_height INT,
    screen_density DOUBLE,
    time_zone VARCHAR(100),
    language VARCHAR(10),
    country_code VARCHAR(10),
    network_type VARCHAR(50),
    carrier_name VARCHAR(100),
    battery_level INT,
    is_charging BOOLEAN,
    total_storage BIGINT,
    available_storage BIGINT,
    ram INT,
    cpu_architecture VARCHAR(50),
    is_tablet BOOLEAN,
    is_emulator BOOLEAN,
    device_name VARCHAR(255),
    unique_id VARCHAR(255),
    scheduled_at TIMESTAMP NULL,
    sent_at TIMESTAMP NULL,
    delivered_at TIMESTAMP NULL,
    read_at TIMESTAMP NULL,
    retry_count INT DEFAULT 0,
    error_message VARCHAR(500),
    priority VARCHAR(20) DEFAULT 'normal',
    sound VARCHAR(100) DEFAULT 'default',
    badge INT DEFAULT 0,
    channel_id VARCHAR(100),
    category_id VARCHAR(100),
    is_silent BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Notification Devices Table

```sql
CREATE TABLE notification_devices (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    expo_push_token VARCHAR(255) UNIQUE NOT NULL,
    device_id VARCHAR(255),
    device_type VARCHAR(50),
    device_brand VARCHAR(100),
    device_model VARCHAR(100),
    os_version VARCHAR(50),
    app_version VARCHAR(50),
    screen_width INT,
    screen_height INT,
    screen_density DOUBLE,
    time_zone VARCHAR(100),
    language VARCHAR(10),
    country_code VARCHAR(10),
    network_type VARCHAR(50),
    carrier_name VARCHAR(100),
    battery_level INT,
    is_charging BOOLEAN,
    total_storage BIGINT,
    available_storage BIGINT,
    ram INT,
    cpu_architecture VARCHAR(50),
    is_tablet BOOLEAN,
    is_emulator BOOLEAN,
    device_name VARCHAR(255),
    unique_id VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Notification Preferences Table

```sql
CREATE TABLE notification_preferences (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT UNIQUE NOT NULL,
    push_enabled BOOLEAN DEFAULT TRUE,
    email_enabled BOOLEAN DEFAULT FALSE,
    sms_enabled BOOLEAN DEFAULT FALSE,
    booking_updates BOOLEAN DEFAULT TRUE,
    worker_assignments BOOLEAN DEFAULT TRUE,
    payment_notifications BOOLEAN DEFAULT TRUE,
    promotional_messages BOOLEAN DEFAULT TRUE,
    system_announcements BOOLEAN DEFAULT TRUE,
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start VARCHAR(5) DEFAULT '22:00',
    quiet_hours_end VARCHAR(5) DEFAULT '08:00',
    max_notifications_per_day INT DEFAULT 50,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## API Endpoints

### Device Management

#### POST /api/v1/notifications/devices

Register a new device for push notifications.

**Request Body:**

```json
{
  "expo_push_token": "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
  "device_info": {
    "device_id": "unique-device-id",
    "device_type": "ios",
    "device_brand": "Apple",
    "device_model": "iPhone 14",
    "os_version": "iOS 17.0",
    "app_version": "1.0.0",
    "screen_width": 390,
    "screen_height": 844,
    "screen_density": 3.0,
    "time_zone": "America/New_York",
    "language": "en",
    "country_code": "US",
    "network_type": "wifi",
    "carrier_name": "Verizon",
    "battery_level": 85,
    "is_charging": false,
    "total_storage": 128000000000,
    "available_storage": 64000000000,
    "ram": 6144,
    "cpu_architecture": "arm64",
    "is_tablet": false,
    "is_emulator": false,
    "device_name": "John's iPhone",
    "unique_id": "unique-device-identifier"
  }
}
```

**Response:**

```json
{
  "message": "Device registered successfully",
  "data": {
    "expo_push_token": "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
    "device_info": { ... }
  }
}
```

#### DELETE /api/v1/notifications/devices/:token

Unregister a device from push notifications.

#### GET /api/v1/notifications/devices

Get all devices for the current user.

### Notification Preferences

#### GET /api/v1/notifications/preferences

Get user's notification preferences.

**Response:**

```json
{
  "message": "Preferences retrieved successfully",
  "data": {
    "id": 1,
    "user_id": 1,
    "push_enabled": true,
    "email_enabled": false,
    "sms_enabled": false,
    "booking_updates": true,
    "worker_assignments": true,
    "payment_notifications": true,
    "promotional_messages": true,
    "system_announcements": true,
    "quiet_hours_enabled": false,
    "quiet_hours_start": "22:00",
    "quiet_hours_end": "08:00",
    "max_notifications_per_day": 50
  }
}
```

#### PUT /api/v1/notifications/preferences

Update user's notification preferences.

**Request Body:**

```json
{
  "push_enabled": true,
  "booking_updates": true,
  "worker_assignments": true,
  "payment_notifications": false,
  "promotional_messages": false,
  "system_announcements": true,
  "quiet_hours_enabled": true,
  "quiet_hours_start": "23:00",
  "quiet_hours_end": "07:00",
  "max_notifications_per_day": 30
}
```

### Notification Management

#### GET /api/v1/notifications

Get user's notifications with pagination.

**Query Parameters:**

- `page`: Page number (default: 1)
- `limit`: Items per page (default: 20)
- `status`: Filter by status (pending, sent, delivered, failed, read)

**Response:**

```json
{
  "message": "Notifications retrieved successfully",
  "data": {
    "notifications": [
      {
        "id": 1,
        "user_id": 1,
        "title": "Booking Confirmed",
        "body": "Your booking has been confirmed",
        "type": "booking_accepted",
        "status": "read",
        "data": "{\"booking_id\": 123}",
        "created_at": "2024-01-01T10:00:00Z",
        "read_at": "2024-01-01T10:05:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 50,
      "pages": 3
    }
  }
}
```

#### GET /api/v1/notifications/:id

Get a specific notification.

#### PUT /api/v1/notifications/:id/read

Mark a notification as read.

#### DELETE /api/v1/notifications/:id

Delete a notification.

#### POST /api/v1/notifications/test

Send a test notification to the current user.

## Device Information Collection

The system collects comprehensive device information:

### Basic Device Info

- **Device ID**: Unique device identifier
- **Device Type**: ios, android, web
- **Device Brand**: Apple, Samsung, etc.
- **Device Model**: iPhone 14, Galaxy S23, etc.
- **OS Version**: iOS 17.0, Android 13, etc.
- **App Version**: 1.0.0

### Display Information

- **Screen Width/Height**: Device screen dimensions
- **Screen Density**: Pixel density (DPI)
- **Is Tablet**: Whether device is a tablet

### System Information

- **RAM**: Available memory in MB
- **CPU Architecture**: arm64, x86_64, etc.
- **Total Storage**: Total device storage in bytes
- **Available Storage**: Available storage in bytes

### Network Information

- **Network Type**: wifi, cellular, none
- **Carrier Name**: Mobile carrier (for cellular)
- **Country Code**: Device country

### Battery Information

- **Battery Level**: Current battery percentage
- **Is Charging**: Whether device is charging

### User Preferences

- **Time Zone**: Device timezone
- **Language**: Device language
- **Device Name**: User-defined device name

### Security Information

- **Is Emulator**: Whether running in emulator
- **Unique ID**: Device-specific unique identifier

## Notification Types

### Booking Notifications

- `booking_created`: New booking created
- `booking_accepted`: Booking accepted by worker
- `booking_in_progress`: Service started
- `booking_completed`: Service completed
- `booking_cancelled`: Booking cancelled

### Worker Notifications

- `worker_assigned`: Worker assigned to booking

### Payment Notifications

- `payment_received`: Payment received

### System Notifications

- `promotion`: Promotional messages
- `system`: System announcements

## Usage Examples

### Sending a Booking Notification

```go
err := utils.SendNotificationAndSave(
    userID,
    "Booking Confirmed",
    "Your booking has been confirmed and a worker will arrive soon.",
    models.NotificationTypeBookingAccepted,
    map[string]interface{}{
        "booking_id": 123,
        "worker_name": "John Doe",
        "estimated_time": "15 minutes",
    },
)
```

### Checking Notification Preferences

```go
shouldSend, err := utils.ShouldSendNotification(userID, models.NotificationTypeBookingUpdates)
if err != nil {
    // Handle error
}

if shouldSend {
    // Send notification
}
```

### Registering a Device

```go
deviceInfo := models.DeviceInfo{
    DeviceID:       "unique-device-id",
    DeviceType:     "ios",
    DeviceBrand:    "Apple",
    DeviceModel:    "iPhone 14",
    OSVersion:      "iOS 17.0",
    AppVersion:     "1.0.0",
    ScreenWidth:    390,
    ScreenHeight:   844,
    ScreenDensity:  3.0,
    TimeZone:       "America/New_York",
    Language:       "en",
    CountryCode:    "US",
    NetworkType:    "wifi",
    BatteryLevel:   85,
    IsCharging:     false,
    IsTablet:       false,
    IsEmulator:     false,
}

err := utils.RegisterDevice(userID, expoPushToken, deviceInfo)
```

## Best Practices

### Frontend Integration

1. **Register Device on App Start**: Register device when app launches
2. **Update Device Info**: Update device info when app comes to foreground
3. **Handle Token Changes**: Re-register device when Expo push token changes
4. **Respect Preferences**: Check user preferences before sending notifications

### Backend Integration

1. **Check Preferences**: Always check user preferences before sending
2. **Handle Failures**: Implement retry logic for failed notifications
3. **Log Everything**: Log all notification attempts for debugging
4. **Rate Limiting**: Respect user's daily notification limits

### Security Considerations

1. **Token Validation**: Validate Expo push tokens
2. **User Authorization**: Ensure users can only manage their own notifications
3. **Data Privacy**: Don't store sensitive information in notification data
4. **Token Rotation**: Handle token expiration and rotation

## Error Handling

### Common Errors

- **Invalid Token**: Expo push token is invalid or expired
- **Device Not Found**: No active devices for user
- **Preferences Disabled**: User has disabled notifications
- **Rate Limited**: Too many notifications sent

### Error Responses

```json
{
  "error": "Notifications disabled",
  "message": "Push notifications are disabled for this user"
}
```

## Monitoring and Analytics

### Key Metrics

- **Delivery Rate**: Percentage of notifications delivered
- **Open Rate**: Percentage of notifications opened
- **Device Distribution**: Distribution across device types
- **Geographic Distribution**: Distribution across countries
- **Error Rates**: Rate of failed notifications

### Logging

All notification events are logged with:

- User ID
- Device information
- Notification type
- Success/failure status
- Error messages (if any)
- Timestamps

This comprehensive notification system provides enterprise-grade push notification capabilities with detailed device tracking and user preference management.
