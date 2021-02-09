#!/bin/sh

set -e

go test
sqlboiler sqlite3 --wipe
echo "all done!"