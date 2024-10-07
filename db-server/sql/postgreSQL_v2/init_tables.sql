-- Users Table
CREATE TABLE at_users (
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
CREATE TABLE at_trips (
    trip_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_name TEXT NOT NULL,
    location TEXT NOT NULL,
    difficulty_level TEXT NOT NULL,  -- Can be changed to ENUM: Easy, Moderate, Difficult
    difficulty_level UUID REFERENCES et_trip_difficulty(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    max_participants INTEGER NOT NULL CHECK (max_participants > 0),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Booking Status Table
CREATE TABLE et_booking_status (
    status_id integer NOT NULL DEFAULT nextval('et_booking_status_id_seq'::regclass), --integer PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL,  -- E.g., Pending, Confirmed, Cancelled
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Bookings Table
CREATE TABLE at_bookings (
    booking_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES at_users(id) ON DELETE CASCADE,
    trip_id UUID REFERENCES at_trips(id) ON DELETE CASCADE,
    booking_status UUID REFERENCES et_booking_status(id),
    booking_date TIMESTAMP DEFAULT NOW(),
    total_cost DECIMAL(10, 2) NOT NULL CHECK (total_cost >= 0),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Trip Features Table (optional)
CREATE TABLE at_trip_features (
    feature_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID REFERENCES at_trips(id) ON DELETE CASCADE,
    feature_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Payments Table (optional)
CREATE TABLE at_payments (
    payment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID REFERENCES at_bookings(id) ON DELETE CASCADE,
    payment_method TEXT NOT NULL,  -- E.g., Credit Card, PayPal
    amount DECIMAL(10, 2) NOT NULL CHECK (amount >= 0),
    payment_status TEXT NOT NULL,  -- E.g., Pending, Completed, Failed
    payment_date TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Trip Reviews Table (optional)
CREATE TABLE at_trip_reviews (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id UUID REFERENCES at_trips(trip_id) ON DELETE CASCADE,
    user_id UUID REFERENCES at_users(user_id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert default booking statuses
INSERT INTO at_booking_status (status_id, status, created_at, updated_at)
VALUES 
    (gen_random_uuid(), 'Pending', NOW(), NOW()),
    (gen_random_uuid(), 'Confirmed', NOW(), NOW()),
    (gen_random_uuid(), 'Cancelled', NOW(), NOW());
