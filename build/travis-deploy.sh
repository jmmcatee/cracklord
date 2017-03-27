#!/usr/bin/env bash

distros = (
  "ubuntu/trusty"
  "debian/jessie"
)

ls

for i in "${distros[@]}"; do
	echo -n "Pushing files to packagecloud for $i..."
	package_cloud push emperorcow/cracklord/$i *.deb
	echo "done"
done