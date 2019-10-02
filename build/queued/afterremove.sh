#!/bin/sh

if [ $1 = "upgrade" ]; then 
	systemctl start cracklord-queued >/dev/null 2>&1 || true
fi
