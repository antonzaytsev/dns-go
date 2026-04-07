#!/bin/sh
# Ensure /logs directory exists and is writable by the dns user (UID 1000)
# This handles the case where a host volume is mounted over /logs
# with permissions that don't allow the dns user to write.
mkdir -p /logs
chown 1000:1000 /logs

# Drop privileges and exec the main process
exec su-exec dns "$@"
