#!/bin/sh

if [ ! -d /var/cracklord ]; then 
	mkdir -p /var/cracklord
fi

if [ ! -d /var/log/cracklord ]; then 
	mkdir -p /var/log/cracklord
fi

if [ -d /etc/cracklord ]; then 
	chown -R cracklord:cracklord /etc/cracklord
fi
if [ -d /var/cracklord ]; then 
	chown -R cracklord:cracklord /var/cracklord
	chmod -R 740 /var/cracklord
fi
if [ -d /var/log/cracklord ]; then 
	chown -R cracklord:cracklord /var/log/cracklord
fi
