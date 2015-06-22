#!/bin/sh

if [ $1 = "upgrade" ]; then 
	service cracklord-queued start >/dev/null 2>&1 || true
fi
