#!/bin/sh

dbName=generated.db

test -f $dbName && rm $dbName
sqlite3 $dbName < schema-v1.sql
sqlboiler sqlite3 --wipe