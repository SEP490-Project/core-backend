-- Migration: Refactor files table with status tracking
-- Created: 2025-11-18
-- Add new columns
ALTER TABLE files
ADD COLUMN IF NOT EXISTS storage_key TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
ADD COLUMN IF NOT EXISTS error_reason TEXT,
ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Create index for soft deletes
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at);

-- Create index for status queries
CREATE INDEX IF NOT EXISTS idx_files_status ON files(status) WHERE deleted_at IS NULL;

-- Create index for storage key lookups
CREATE INDEX IF NOT EXISTS idx_files_storage_key ON files(storage_key) WHERE deleted_at IS NULL;

-- Backfill storage_key for existing records (URL path extraction)
UPDATE files
SET storage_key = SUBSTRING(url FROM '([^/]+/[^/]+)$')
WHERE storage_key = '' AND url IS NOT NULL;

-- Make URL nullable (will be populated after upload completes)
ALTER TABLE files ALTER COLUMN url DROP NOT NULL;

