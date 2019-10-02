#!/bin/sh

systemctl stop cracklord-resourced >/dev/null 2>&1 || true

if [ $1 = "remove" ]; then
	if getent passwd cracklord >/dev/null ; then
		userdel cracklord
	fi

	if getent group cracklord >/dev/null ; then
		groupdel cracklord
	fi
fi
