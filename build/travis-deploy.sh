#!/usr/bin/env bash

echo "Deployment starting"


distros=( 
  "ubuntu/bionic" 
  "ubuntu/trusty" 
  "ubuntu/xenial"
  "debian/jessie" 
)

for i in "${distros[@]}"; do
	package_cloud push emperorcow/cracklord/$i *.deb
done