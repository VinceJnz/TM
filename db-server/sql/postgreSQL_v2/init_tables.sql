-- Users Table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone_number TEXT,
    user_status TEXT NOT NULL DEFAULT 'Active',  -- You could use an enum here if needed
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Trips Table
CREATE TABLE trips (
    trip_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_name TEXT NOT NULL,
    location TEXT NOT NULL,
    difficulty_level TEXT NOT NULL,  -- Can be changed to ENUM: Easy, Moderate, Difficult
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    max_participants INTEGER NOT NULL CHECK (max_participants > 0),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Booking Status Table
CREATE TABLE booking_status (
    status_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL,  -- E.g., Pending, Confirmed, Cancelled
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Bookings Table
CREATE TABLE bookings (
    booking_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    trip_id UUID REFERENCES trips(trip_id) ON DELETE CASCADE,
    booking_status UUID REFERENCES booking_status(status_id),
    booking_date TIMESTAMP DEFAULT NOW(),
    total_cost DECIMAL(10, 2) NOT NULL CHECK (total_cost >= 0),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Trip Features Table (optional)
CREATE TABLE trip_features (
    feature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID REFERENCES trips(trip_id) ON DELETE CASCADE,
    feature_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Payments Table (optional)
CREATE TABLE payments (
    payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID REFERENCES bookings(booking_id) ON DELETE CASCADE,
    payment_method TEXT NOT NULL,  -- E.g., Credit Card, PayPal
    amount DECIMAL(10, 2) NOT NULL CHECK (amount >= 0),
    payment_status TEXT NOT NULL,  -- E.g., Pending, Completed, Failed
    payment_date TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Trip Reviews Table (optional)
CREATE TABLE trip_reviews (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID REFERENCES trips(trip_id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(user_id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert default booking statuses
INSERT INTO booking_status (status_id, status, created_at, updated_at)
VALUES 
    (gen_random_uuid(), 'Pending', NOW(), NOW()),
    (gen_random_uuid(), 'Confirmed', NOW(), NOW()),
    (gen_random_uuid(), 'Cancelled', NOW(), NOW());
