#!/usr/bin/env bash

distros = (
  "ubuntu/trusty"
  "debian/jessie"
)

for i in "${distros[@]}"; do
	package_cloud push emperorcow/cracklord/$i *.deb
done