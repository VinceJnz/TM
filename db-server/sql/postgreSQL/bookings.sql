CREATE TABLE IF NOT EXISTS at_bookings (
  ID SERIAL PRIMARY KEY,
  Owner_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
  Notes TEXT,
  From_date TIMESTAMP DEFAULT NULL,
  To_date TIMESTAMP DEFAULT NULL,
  Booking_status_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
  Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.Modified = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_at_bookings_modified
BEFORE UPDATE ON at_bookings
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();