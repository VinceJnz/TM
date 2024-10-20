-- Created by CoPilot 20241020

-- Table for user types
CREATE TABLE user_types (
    user_type_id SERIAL PRIMARY KEY,
    user_type_name VARCHAR(255) NOT NULL -- Example: 'infant', 'child', 'youth', 'adult'
);

-- Table for user information
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    user_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL,
    user_type_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_type_id) REFERENCES user_types(user_type_id)
);

-- Table for seasons
CREATE TABLE seasons (
    season_id SERIAL PRIMARY KEY,
    season_name VARCHAR(255) NOT NULL -- Example: 'summer', 'winter', 'off-peak'
);

-- Table for trip types
CREATE TABLE trip_types (
    trip_type_id SERIAL PRIMARY KEY,
    trip_type_name VARCHAR(255) NOT NULL -- Example: 'hiking', 'camping', 'rafting'
);

-- Table for trip information
CREATE TABLE trips (
    trip_id SERIAL PRIMARY KEY,
    trip_name VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    difficulty VARCHAR(50) NOT NULL,
    distance DECIMAL(5, 2) NOT NULL, -- in kilometers
    duration DECIMAL(5, 2) NOT NULL, -- in hours
    trip_type_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_type_id) REFERENCES trip_types(trip_type_id)
);

-- Table for booking information
CREATE TABLE bookings (
    booking_id SERIAL PRIMARY KEY,
    trip_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    booking_date DATE NOT NULL,
    from_date DATE NOT NULL,
    to_date DATE NOT NULL,
    group_booking_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (trip_id) REFERENCES trips(trip_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (group_booking_id) REFERENCES group_bookings(group_booking_id)
);

-- Table for group bookings
CREATE TABLE group_bookings (
    group_booking_id SERIAL PRIMARY KEY,
    group_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing trip costs with user types, seasons, and trip types
CREATE TABLE trip_costs (
    cost_id SERIAL PRIMARY KEY,
    trip_id INTEGER NOT NULL,
    user_type_id INTEGER NOT NULL,
    season_id INTEGER NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    FOREIGN KEY (trip_id) REFERENCES trips(trip_id),
    FOREIGN KEY (user_type_id) REFERENCES user_types(user_type_id),
    FOREIGN KEY (season_id) REFERENCES seasons(season_id)
);

-- Table for user payments
CREATE TABLE user_payments (
    payment_id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    booking_id INTEGER NOT NULL,
    payment_date DATE NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    payment_method VARCHAR(255), -- Example: 'credit_card', 'paypal', etc.
    FOREIGN KEY (user_id) REFERENCES users(user_id),
    FOREIGN KEY (booking_id) REFERENCES bookings(booking_id)
);

-- Insert sample data into user_types
INSERT INTO user_types (user_type_name)
VALUES 
('infant'),
('child'),
('youth'),
('adult');

-- Insert sample data into seasons
INSERT INTO seasons (season_name)
VALUES 
('summer'),
('winter'),
('off-peak');

-- Insert sample data into trip_types
INSERT INTO trip_types (trip_type_name)
VALUES 
('hiking'),
('camping'),
('rafting');

-- Insert sample data into users
INSERT INTO users (user_name, email, status, user_type_id)
VALUES 
('John Doe', 'john@example.com', 'active', 4),
('Jane Smith', 'jane@example.com', 'active', 4);

-- Insert sample data into trips
INSERT INTO trips (trip_name, location, difficulty, distance, duration, trip_type_id)
VALUES 
('Mountain Trail', 'Mountain Range', 'Hard', 10.5, 8.0, 1),
('Forest Walk', 'Dense Forest', 'Moderate', 5.0, 3.0, 1);

-- Insert sample data into bookings with group_booking_id
INSERT INTO bookings (trip_id, user_id, booking_date, from_date, to_date, group_booking_id)
VALUES 
(1, 1, '2024-11-01', '2024-11-05', '2024-11-10', 1),
(2, 2, '2024-11-05', '2024-11-08', '2024-11-12', 2);

-- Insert sample data into group_bookings
INSERT INTO group_bookings (group_name)
VALUES ('Family Trip'), ('Corporate Retreat');

-- Insert sample data into trip_costs
INSERT INTO trip_costs (trip_id, user_type_id, season_id, amount)
VALUES 
(1, 4, 1, 200.00), -- Adult, Summer
(1, 3, 1, 150.00), -- Youth, Summer
(1, 2, 1, 100.00), -- Child, Summer
(2, 4, 2, 120.00), -- Adult, Winter
(2, 3, 2, 90.00);  -- Youth, Winter

-- Insert sample data into user_payments
INSERT INTO user_payments (user_id, booking_id, payment_date, amount, payment_method)
VALUES 
(1, 1, '2024-11-01', 200.00, 'credit_card'),
(2, 2, '2024-11-05', 90.00, 'paypal');

-- Query to calculate total trip cost based on user type, season, and trip type
SELECT 
    u.user_name,
    t.trip_name,
    ut.user_type_name,
    s.season_name,
    SUM(tc.amount) AS total_trip_cost
FROM 
    bookings b
    JOIN users u ON b.user_id = u.user_id
    JOIN trips t ON b.trip_id = t.trip_id
    JOIN user_types ut ON u.user_type_id = ut.user_type_id
    JOIN seasons s ON tc.season_id = s.season_id
    JOIN trip_costs tc ON b.trip_id = tc.trip_id
GROUP BY 
    u.user_name, t.trip_name, ut.user_type_name, s.season_name;

-- Query to calculate the amount each user has paid
SELECT 
    u.user_name, 
    b.booking_id, 
    SUM(up.amount) AS total_paid
FROM 
    user_payments up
    JOIN users u ON up.user_id = u.user_id
    JOIN bookings b ON up.booking_id = b.booking_id
GROUP BY 
    u.user_name, b.booking_id;
