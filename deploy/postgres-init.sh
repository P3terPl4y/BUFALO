#!/bin/sh
set -eu
# This runs only when PostgreSQL initializes a new, empty data volume.
psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" --set=ON_ERROR_STOP=1 <<'SQL'
\getenv app_user APP_DB_USER
\getenv app_password APP_DB_PASSWORD
\getenv migration_user MIGRATION_DB_USER
\getenv migration_password MIGRATION_DB_PASSWORD
\getenv database POSTGRES_DB
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 32', :'app_user', :'app_password') \gexec
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE CONNECTION LIMIT 8', :'migration_user', :'migration_password') \gexec
ALTER DATABASE :"database" OWNER TO :"migration_user";
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO :"migration_user";
GRANT CONNECT ON DATABASE :"database" TO :"app_user";
GRANT USAGE ON SCHEMA public TO :"app_user";
ALTER DEFAULT PRIVILEGES FOR ROLE :"migration_user" IN SCHEMA public GRANT SELECT,INSERT,UPDATE,DELETE ON TABLES TO :"app_user";
ALTER DEFAULT PRIVILEGES FOR ROLE :"migration_user" IN SCHEMA public GRANT USAGE,SELECT ON SEQUENCES TO :"app_user";
SQL
