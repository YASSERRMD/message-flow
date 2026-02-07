#!/bin/sh
set -e

until pg_isready -h db -U postgres; do
  sleep 1
done

for file in /migrations/[0-9][0-9][0-9]_*.sql; do
  echo "Applying migration: $file"
  psql "$DATABASE_URL" -f "$file"
done
