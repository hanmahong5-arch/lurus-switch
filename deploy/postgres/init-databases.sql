-- Lurus Switch PostgreSQL Database Initialization
-- Creates all required databases for microservices

-- NEW-API Database
CREATE DATABASE new_api;
GRANT ALL PRIVILEGES ON DATABASE new_api TO lurus;

-- Provider Service Database
CREATE DATABASE lurus_provider;
GRANT ALL PRIVILEGES ON DATABASE lurus_provider TO lurus;

-- Billing Service Database
CREATE DATABASE lurus_billing;
GRANT ALL PRIVILEGES ON DATABASE lurus_billing TO lurus;

-- Sync Service Database
CREATE DATABASE lurus_sync;
GRANT ALL PRIVILEGES ON DATABASE lurus_sync TO lurus;

-- Subscription Service Database (future)
CREATE DATABASE lurus_subscription;
GRANT ALL PRIVILEGES ON DATABASE lurus_subscription TO lurus;

-- Log for initialization
DO $$
BEGIN
    RAISE NOTICE 'Lurus Switch databases initialized successfully';
END $$;
