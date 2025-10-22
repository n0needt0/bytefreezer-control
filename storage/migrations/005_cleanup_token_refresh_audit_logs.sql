-- Migration: Cleanup token_refresh audit logs
-- Description: Remove noisy token_refresh entries from audit logs
-- Date: 2025-10-21

-- Up migration: Delete all token_refresh audit log entries
-- These are too noisy and were removed from logging in v2.2.1
DELETE FROM control_audit_log WHERE action = 'token_refresh';
