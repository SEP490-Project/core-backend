-- Migration: Create system_alerts and alert_acknowledgments tables for centralized alert management

-- Create alert type enum
DO $$ BEGIN
    CREATE TYPE alert_type AS ENUM ('WARNING', 'ERROR', 'INFO');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;

-- Create alert category enum
DO $$ BEGIN
    CREATE TYPE alert_category AS ENUM (
        -- Content Staff Alerts
        'CONTENT_REJECTED', 'LOW_CTR', 'LOW_ENGAGEMENT', 'SCHEDULE_FAILED',
        'PENDING_APPROVAL', 'DEADLINE_APPROACHING',
        -- Marketing Staff Alerts
        'CAMPAIGN_DEADLINE', 'BUDGET_EXCEEDED',
        -- Sales Staff Alerts
        'ORDER_ISSUE', 'PAYMENT_OVERDUE',
        -- Admin Alerts
        'SYSTEM_HEALTH', 'SECURITY_ISSUE'
        );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;

-- Create alert severity enum
DO $$ BEGIN
    CREATE TYPE alert_severity AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;

-- Create alert status enum
DO $$ BEGIN
    CREATE TYPE alert_status AS ENUM ('ACTIVE', 'RESOLVED', 'EXPIRED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;

-- Create system_alerts table
CREATE TABLE IF NOT EXISTS system_alerts (
id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
type VARCHAR (30) NOT NULL,
category VARCHAR (50) NOT NULL,
severity VARCHAR (20) NOT NULL DEFAULT 'MEDIUM',
title VARCHAR (255) NOT NULL,
description TEXT NOT NULL,
metadata JSONB DEFAULT '{}',
target_roles JSONB NOT NULL DEFAULT '[]',
reference_id UUID,
reference_type VARCHAR (50),
action_url TEXT,
status VARCHAR (20) NOT NULL DEFAULT 'ACTIVE',
acknowledgement JSONB DEFAULT '{}',
resolved_by UUID,
resolved_at TIMESTAMP WITH TIME ZONE,
expires_at TIMESTAMP WITH TIME ZONE,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW (),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW ()
) ;


-- Create indexes for system_alerts
CREATE INDEX IF NOT EXISTS idx_system_alerts_status ON system_alerts (status) ;
CREATE INDEX IF NOT EXISTS idx_system_alerts_type ON system_alerts (type) ;
CREATE INDEX IF NOT EXISTS idx_system_alerts_category ON system_alerts (category) ;
CREATE INDEX IF NOT EXISTS idx_system_alerts_severity ON system_alerts (severity) ;
CREATE INDEX IF NOT EXISTS idx_system_alerts_expires_at ON system_alerts (expires_at) ;
CREATE INDEX IF NOT EXISTS idx_system_alerts_created_at ON system_alerts (created_at DESC) ;
CREATE INDEX IF NOT EXISTS idx_system_alerts_reference ON system_alerts (reference_id,
reference_type) ;

-- GIN index for JSONB target_roles (for role-based filtering)
CREATE INDEX IF NOT EXISTS idx_system_alerts_target_roles ON system_alerts USING GIN (target_roles) ;

-- Comments
COMMENT ON TABLE system_alerts IS 'Centralized alert system for all staff roles' ;
COMMENT ON COLUMN system_alerts.type IS 'Alert type: WARNING, ERROR, INFO' ;
COMMENT ON COLUMN system_alerts.category IS 'Alert category for filtering and grouping' ;
COMMENT ON COLUMN system_alerts.severity IS 'Alert severity: LOW, MEDIUM, HIGH, CRITICAL' ;
COMMENT ON COLUMN system_alerts.target_roles IS 'JSON array of user roles that should see this alert' ;
COMMENT ON COLUMN system_alerts.reference_id IS 'Optional reference to related entity (content, task, campaign, etc.)' ;
COMMENT ON COLUMN system_alerts.reference_type IS 'Type of the referenced entity' ;
COMMENT ON COLUMN system_alerts.action_url IS 'URL to navigate when alert is clicked' ;
COMMENT ON COLUMN system_alerts.expires_at IS 'Optional expiration time for auto-expiring alerts' ;
