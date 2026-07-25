CREATE ROLE skawld_owner LOGIN PASSWORD 'skawld_owner_dev';
CREATE ROLE skawld_app LOGIN PASSWORD 'skawld_app_dev' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE DATABASE skawld OWNER skawld_owner;

CREATE ROLE keycloak LOGIN PASSWORD 'keycloak_dev';
CREATE DATABASE keycloak OWNER keycloak;

\connect skawld
CREATE EXTENSION vector;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT CONNECT ON DATABASE skawld TO skawld_app;
GRANT USAGE ON SCHEMA public TO skawld_app;
