-- Script to reset the setup process
-- This will allow you to go through the setup wizard again
-- WARNING: This will delete all users and setup-related data

-- Delete setup completion flag
DELETE FROM settings WHERE key = 'setup.completed';

-- Delete setup progress tracking
DELETE FROM settings WHERE key LIKE 'setup.step.%';

-- Delete all users (required for setup to be accessible)
DELETE FROM users;

-- Optional: Clear all sessions (if table exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'sessions') THEN
        DELETE FROM sessions;
    END IF;
END $$;

SELECT 'Setup has been reset. All users have been deleted. You can now access the setup wizard.' AS message;
