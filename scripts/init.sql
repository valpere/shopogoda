-- Initialize the ShoPogoda database with extensions and initial data

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable PostGIS extension for location data (optional)
-- CREATE EXTENSION IF NOT EXISTS postgis;

-- Create indexes for better performance
-- These will be created by GORM migrations, but we can add custom ones here

-- Example: Add custom indexes
-- CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_weather_data_timestamp_location
-- ON weather_data (timestamp DESC, location_id);

-- Create initial admin user (optional)
-- INSERT INTO users (id, username, first_name, role, is_active, created_at, updated_at)
-- VALUES (123456789, 'admin', 'Admin', 3, true, NOW(), NOW())
-- ON CONFLICT (id) DO NOTHING;

-- Insert some sample data
INSERT INTO users (id, username, first_name, role, is_active, created_at, updated_at)
VALUES
(1, 'testuser', 'Test User', 1, true, NOW(), NOW()),
(2, 'moderator', 'Moderator User', 2, true, NOW(), NOW()),
(3, 'admin', 'Admin User', 3, true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
