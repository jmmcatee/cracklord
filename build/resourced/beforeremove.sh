#!/bin/sh

service cracklord-resourced stop >/dev/null 2>&1

if [ $1 = "remove" ]; then
	if getent passwd cracklord >/dev/null ; then
		userdel cracklord
	fi

	if getent group cracklord >/dev/null ; then
		groupdel cracklord
	fi
fi

rm -r /var/cracklord