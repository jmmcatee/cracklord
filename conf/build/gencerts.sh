#!/bin/bash
# Generate CA Key
openssl genrsa -out cracklord_ca.key 4096

# General CA Certificate
openssl req -x509 -new -nodes -key cracklord_ca.key -days 1024 -out cracklord_ca.pem -config cracklord_ca_ssl.conf -batch

# General QUEUED Key, Request, & Certificate
openssl genrsa -out queued.key 4096
openssl req -new -key queued.key -out queued.csr -config cracklord_queued_ssl.conf -batch
openssl x509 -req -in queued.csr -CA cracklord_ca.pem -CAkey cracklord_ca.key -CAcreateserial -out queued.crt -days 500

# General RESOURCED Key, Request, & Certificate
openssl genrsa -out resourced.key 4096
openssl req -new -key resourced.key -out resourced.csr -config cracklord_resourced_ssl.conf -batch
openssl x509 -req -in resourced.csr -CA cracklord_ca.pem -CAkey cracklord_ca.key -CAcreateserial -out resourced.crt -days 500

# Remove requests
rm -rf *.csr