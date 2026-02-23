#!/usr/bin/env bash

envsubst < /docker-entrypoint-initdb.d/01-create-dbs.sql | \
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"