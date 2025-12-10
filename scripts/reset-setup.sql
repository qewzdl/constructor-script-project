-- Script to reset the setup process
-- This will allow you to go through the setup wizard again
-- WARNING: This will delete all users and setup-related data

-- Delete setup completion flag
DELETE FROM settings WHERE key = 'setup.completed';

-- Delete setup progress tracking
DELETE FROM settings WHERE key LIKE 'setup.step.%';

-- Delete all users (required for setup to be accessible)
DELETE FROM users;

-- Delete setup progress data (for stepwise setup)
DO $$ 
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'setup_progresses') THEN
        DELETE FROM setup_progresses;
    END IF;
END $$;

-- Optional: Clear all sessions (if table exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'sessions') THEN
        DELETE FROM sessions;
    END IF;
END $$;

SELECT 'Setup has been reset. All users and setup progress have been deleted. You can now access the setup wizard.' AS message;
