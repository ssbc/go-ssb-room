#!/bin/sh

set -e

go test
sqlboiler sqlite3 --wipe --no-tests
echo "all done!"
