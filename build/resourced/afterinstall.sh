#!/bin/bash

mkdir -p /var/cracklord
addgroup --system cracklord
adduser --system --disabled-password --no-create-home --disabled-login cracklord
chown -R cracklord:cracklord /etc/cracklord
chown -R cracklord:cracklord /var/cracklord