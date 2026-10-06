#!/bin/sh
get() {
	sed -n "s/.*\"$1\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p" /config.json | head -n1
}

export POSTGRES_USER="$(get db_user)"
export POSTGRES_PASSWORD="$(get db_password)"
export POSTGRES_DB="$(get db_name)"

if [ -z "$POSTGRES_USER" ] || [ -z "$POSTGRES_PASSWORD" ] || [ -z "$POSTGRES_DB" ]; then
	echo "db_user, db_password and db_name must be set in config.json" >&2
	exit 1
fi

exec docker-entrypoint.sh postgres