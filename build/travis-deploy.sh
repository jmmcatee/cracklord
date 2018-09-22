#!/usr/bin/env bash

echo "Deployment starting"

gem install package_cloud

distros=( 
  "ubuntu/trusty" 
  "ubuntu/xenial"
  "debian/jessie" 
)

for i in "${distros[@]}"; do
	package_cloud push emperorcow/cracklord/$i *.deb
done