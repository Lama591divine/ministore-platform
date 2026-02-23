\set ON_ERROR_STOP on

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', '${AUTH_USER}', '${AUTH_PASS}')
    WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${AUTH_USER}')
    \gexec

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', '${CATALOG_USER}', '${CATALOG_PASS}')
    WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${CATALOG_USER}')
    \gexec

SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', '${ORDER_USER}', '${ORDER_PASS}')
    WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${ORDER_USER}')
    \gexec

SELECT format('CREATE DATABASE %I OWNER %I', '${AUTH_DB}', '${AUTH_USER}')
    WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '${AUTH_DB}')
    \gexec

SELECT format('CREATE DATABASE %I OWNER %I', '${CATALOG_DB}', '${CATALOG_USER}')
    WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '${CATALOG_DB}')
    \gexec

SELECT format('CREATE DATABASE %I OWNER %I', '${ORDER_DB}', '${ORDER_USER}')
    WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '${ORDER_DB}')
    \gexec