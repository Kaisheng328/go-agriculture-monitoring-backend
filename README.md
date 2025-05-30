# Go-Soil-Moisture-Backend
## Overview

The Go-Soil-Moisture-Backend is a backend service designed to monitor and manage soil moisture data for agricultural purposes. Built using Go, it provides APIs to collect, process, and retrieve soil moisture information, enabling efficient decision-making for irrigation and crop management.

## Features

- **Data Collection**: Accepts soil moisture data from IoT devices or sensors.
- **Data Processing**: Processes and stores data in a structured format for easy retrieval.
- **API Endpoints**: Provides RESTful APIs for accessing soil moisture data.
- **Scalability**: Designed to handle large datasets and multiple concurrent requests.
- **Extensibility**: Easily extendable to include additional features like weather integration or predictive analytics.

## Installation

1. Clone the repository:
    ```bash
    git clone https://github.com/yourusername/go-agriculture-monitoring-backend.git
    ```
2. Navigate to the project directory:
    ```bash
    cd go-agriculture-monitoring-backend
    ```
3. Build the project:
    ```bash
    go build
    ```
4. Run the application:
    ```bash
    ./go-agriculture-monitoring-backend
    ```

## API Endpoints
- **POST /signup**: Register a new user account.
- **POST /login**: Authenticate and log in a user.
- **POST /toggle-ai**: Enable or disable AI features.

### Protected Routes (Require Authentication)
- **POST /promote-admin**: Promote a user to admin role.
- **POST /promote-user**: Demote an admin to user role.
- **POST /sensor-data**: Submit sensor data to the server.
- **GET /history**: Retrieve historical data.
- **GET /users**: Fetch a list of all users.
- **GET /profile**: Retrieve the profile of the logged-in user.
- **GET /abnormal-count**: Get the count of abnormal events.
- **GET /abnormal-history**: Retrieve the history of abnormal events.
- **GET /download-csv**: Download data in CSV format.
- **GET /ws**: Establish a WebSocket connection.
- **PUT /update/{id}**: Update a specific record by ID.
- **DELETE /delete/{id}**: Delete a specific record by ID.
- **DELETE /delete/all**: Delete all records.
- **POST /location**: Submit device location data.
- **GET /get-location/{device_id}**: Retrieve the location of a specific device.

## Configuration
The application can be configured using environment variables:
- `DATABASE_URL`: Full database connection string. Example:
    ```plaintext
    DATABASE_URL=postgresql://username:password@postgresql_url:port/postgres
    ```
Ensure that the `DATABASE_URL` is correctly set to connect to your PostgreSQL database.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request with your changes.

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.