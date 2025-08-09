#!/usr/bin/env bash

# Reference: https://github.com/ory/keto/blob/master/scripts/test-resetdb.sh

set -xeuo pipefail

docker container rm -f test_postgres || true

postgres_port="$(docker container port \
                "$(docker container run --name test_postgres -e "POSTGRES_PASSWORD=secret" -e "POSTGRES_DB=postgres" -p 0.0.0.0:0:5432 -d postgres:17)" 5432 | \
                    head -1 | cut -d: -f2)"

TEST_DATABASE_DSN="postgres://postgres:secret@localhost:${postgres_port}/postgres"
export TEST_DATABASE_DSN

set +e
set +u
set +x
set +o pipefail
