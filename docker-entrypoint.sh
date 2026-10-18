#!/bin/sh
set -e

PB=/usr/local/bin/pocketbase

"$PB" migrate up \
  --dir="/app/pb_data" \
  --migrationsDir="/app/pb_migrations" \
  --hooksDir="/app/pb_hooks"

exec "$PB" serve \
  --http="0.0.0.0:8090" \
  --dir="/app/pb_data" \
  --publicDir="/app/pb_public" \
  --migrationsDir="/app/pb_migrations" \
  --hooksDir="/app/pb_hooks" \
  --automigrate=false
