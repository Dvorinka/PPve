# Admin Login System

This document provides information about the admin login system for the PP Kunovice web application.

## Default Admin Credentials

- **Username**: `admin`
- **Password**: `admin123`

**Important**: Change the default password after the first login in a production environment.

## Accessing the Admin Panel

1. Navigate to `/admin` in your web browser
2. Enter the admin credentials
3. After successful login, you'll be redirected to the admin dashboard

## API Endpoints

### Login
- **URL**: `/api/login`
- **Method**: `POST`
- **Content-Type**: `application/json`
- **Request Body**:
  ```json
  {
    "username": "admin",
    "password": "admin123"
  }
  ```
- **Success Response**:
  - **Code**: 200 OK
  - **Content**:
    ```json
    {
      "token": "jwt.token.here"
    }
    ```
- **Error Response**:
  - **Code**: 401 Unauthorized
  - **Content**:
    ```json
    {
      "error": "Invalid credentials"
    }
    ```

### Protected Endpoints

All protected endpoints require a valid JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

## Environment Variables

- `JWT_SECRET`: Secret key used to sign JWT tokens (default: auto-generated)
- `PORT`: Port the server listens on (default: 80)

## Security Notes

1. Always use HTTPS in production
2. Change the default admin password
3. Set a strong `JWT_SECRET` environment variable in production
4. Consider implementing rate limiting for login attempts
5. Keep the server and dependencies up to date

## Development

To run the server in development mode:

```bash
go run .
```

The admin interface will be available at `http://localhost/admin`
