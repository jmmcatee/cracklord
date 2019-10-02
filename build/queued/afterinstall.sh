#!/bin/sh
CONFDIR="/etc/cracklord"
SSLDIR="$CONFDIR/ssl"

if [ -f $SSLDIR/cracklord_ca_ssl.conf -a -f $SSLDIR/cracklord_queued_ssl.conf -a -f $SSLDIR/cracklord_resourced_ssl.conf ]; then
	if [ ! -f $SSLDIR/cracklord_ca.pem ]; then 
		# Generate the SSL CA to sign all resources to allow authentication
		openssl genrsa -out /etc/cracklord/ssl/cracklord_ca.key 4096
		# General CA Certificate
		openssl req -x509 -new -nodes -key /etc/cracklord/ssl/cracklord_ca.key -days 1024 -out /etc/cracklord/ssl/cracklord_ca.pem -config /etc/cracklord/ssl/cracklord_ca_ssl.conf -batch
	fi

	if [ ! -f $SSLDIR/queued.crt ]; then
		# General QUEUED Key, Request, & Certificate
		openssl genrsa -out /etc/cracklord/ssl/queued.key 4096
		openssl req -new -key /etc/cracklord/ssl/queued.key -out /etc/cracklord/ssl/queued.csr -config /etc/cracklord/ssl/cracklord_queued_ssl.conf -batch
		openssl x509 -req -extensions client_server_ssl -extfile /etc/cracklord/ssl/cracklord_queued_ext.conf -in /etc/cracklord/ssl/queued.csr -CA /etc/cracklord/ssl/cracklord_ca.pem -CAkey /etc/cracklord/ssl/cracklord_ca.key -CAcreateserial -out /etc/cracklord/ssl/queued.crt -days 500
	fi

	if [ ! -f $SSLDIR/resourced.crt ]; then
		# General RESOURCED Key, Request, & Certificate
		openssl genrsa -out /etc/cracklord/ssl/resourced.key 4096
		openssl req -new -key /etc/cracklord/ssl/resourced.key -out /etc/cracklord/ssl/resourced.csr -config /etc/cracklord/ssl/cracklord_resourced_ssl.conf -batch
		openssl x509 -req -in /etc/cracklord/ssl/resourced.csr -CA /etc/cracklord/ssl/cracklord_ca.pem -CAkey /etc/cracklord/ssl/cracklord_ca.key -CAcreateserial -out /etc/cracklord/ssl/resourced.crt -days 500
	fi

	# Remove requests and config files
	rm -r /etc/cracklord/ssl/*.csr >/dev/null 2>&1 || true
fi 

# Set to startup on boot
if [ ! -f /etc/systemd/system/cracklord-queued.service ]; then
	systemctl enable cracklord-queued
fi

systemctl restart cracklord-queued