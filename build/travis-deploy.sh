#!/usr/bin/env bash

echo "Deployment starting"


distros=( 
  "ubuntu/trusty" 
  "debian/jessie" 
)

for i in "${distros[@]}"; do
	package_cloud push emperorcow/cracklord/$i *.deb
done