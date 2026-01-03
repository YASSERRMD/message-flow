#!/bin/sh
set -e

until pg_isready -h db -U postgres; do
  sleep 1
done

psql "$DATABASE_URL" -f /migrations/001_init.sql
psql "$DATABASE_URL" -f /migrations/002_phase2_llm.sql
psql "$DATABASE_URL" -f /migrations/003_phase3_llm_management.sql
psql "$DATABASE_URL" -f /migrations/004_phase3_llm_provider_fields.sql
