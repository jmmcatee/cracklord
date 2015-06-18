#!/bin/sh

# Create directories
mkdir -p /var/cracklord
mkdir -p /var/log/cracklord
chown -R cracklord:cracklord /etc/cracklord
chown -R cracklord:cracklord /var/cracklord
chmod -R 640 /var/cracklord
chown -R cracklord:cracklord /var/log/cracklord

