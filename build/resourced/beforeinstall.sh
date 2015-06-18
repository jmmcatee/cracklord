#!/bin/sh

# create group
if ! getent group cracklord >/dev/null; then
	addgroup --system cracklord
fi

# create user
if ! getent passwd cracklord >/dev/null; then
	adduser --system --disabled-password --no-create-home --disabled-login cracklord
fi