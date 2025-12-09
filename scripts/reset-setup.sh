#!/bin/bash

# Script to reset setup to initial state
# This allows you to go through the setup wizard again

set -e

# Read database credentials from .env file if it exists
if [ -f .env ]; then
    DB_HOST=$(grep '^DB_HOST=' .env | cut -d '=' -f2-)
    DB_PORT=$(grep '^DB_PORT=' .env | cut -d '=' -f2-)
    DB_USER=$(grep '^DB_USER=' .env | cut -d '=' -f2-)
    DB_PASSWORD=$(grep '^DB_PASSWORD=' .env | cut -d '=' -f2-)
    DB_NAME=$(grep '^DB_NAME=' .env | cut -d '=' -f2-)
    PORT=$(grep '^PORT=' .env | cut -d '=' -f2-)
fi

# Use environment variables or defaults
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-devuser}
DB_PASSWORD=${DB_PASSWORD:-devpassword}
DB_NAME=${DB_NAME:-constructor}
PORT=${PORT:-8081}

echo "=========================================="
echo "   Reset Setup Script"
echo "=========================================="
echo ""
echo "This will reset the setup status, allowing you to"
echo "go through the setup wizard again."
echo ""
echo "WARNING: This will DELETE ALL USERS and sessions!"
echo "All other data (posts, pages, etc.) will be preserved."
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled."
    exit 1
fi

echo ""
echo "Resetting setup..."
echo ""

# Execute the SQL script
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f scripts/reset-setup.sql

echo ""
echo "=========================================="
echo "   Setup has been reset!"
echo "=========================================="
echo ""
echo "You can now access the setup wizard at:"
echo "http://localhost:${PORT:-8081}/setup"
echo ""
