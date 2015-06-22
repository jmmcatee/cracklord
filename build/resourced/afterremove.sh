#!/bin/sh

if [ $1 = "remove" ]; then
	if [ -d /var/cracklord ]; then
		rm -r /var/cracklord
	fi
fi

if [ $1 = "upgrade" ]; then 
	service cracklord-resourced start >/dev/null 2>&1 || true
fi