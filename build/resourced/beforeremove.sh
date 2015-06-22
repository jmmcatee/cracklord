#!/bin/sh

service cracklord-resourced stop >/dev/null 2>&1 || true

if getent passwd cracklord >/dev/null ; then
	userdel cracklord
fi

if getent group cracklord >/dev/null ; then
	groupdel cracklord
fi

if [ -d /var/cracklord ]; then
	rm -r /var/cracklord
fi
