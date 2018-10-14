#!/usr/bin/env ruby

# This script goes through the current packages on packagecloud and deletes the older ones
# if there are more that 2 for a distro.  Note that packagecloud differentiates between
# the queue and resource server, so that should give us a total of 4 packages per distro.

require 'json'
require 'pp'
require 'rest-client'
require 'time'

# Takes the name of a package and distribution and determines if we have too many.  If so
# it deletes them. 
def processPackage(distName, pkgName) 
  url = BASE_URL + "/package/deb/#{distName}/#{pkgName}/amd64/versions.json"
  begin
    versions = RestClient.get(url)
  rescue StandardError => msg
    puts msg
    return
  end

  parsed_versions = JSON.parse(versions)
  sorted_versions = parsed_versions.sort_by { |x| Time.parse(x["created_at"]) }

  puts "[*] #{distName} - #{pkgName} - #{sorted_versions.size} existing packages."

  if sorted_versions.size >= LIMIT
    to_yank = sorted_versions.first

    distro_version = to_yank["distro_version"]
    filename = to_yank["filename"]
    yank_url = "/#{distro_version}/#{filename}"
    url = BASE_URL + yank_url

    result = RestClient.delete(url)
    if result == {}
      puts "[!] Successfully yanked #{filename} to make room for new deployment."
    end
  end
end

# Get our environment variables all set.
API_TOKEN=ENV["PACKAGECLOUD_TOKEN"]
USER = 'emperorcow'
REPOSITORY = 'cracklord'
LIMIT = 2    
BASE_URL = "https://#{API_TOKEN}:@packagecloud.io/api/v1/repos/#{USER}/#{REPOSITORY}"
DISTROS = [
  'ubuntu/trusty',
  'ubuntu/xenial',
  'debian/jessie'
]

PACKAGES = [
  'cracklord-queued',
  'cracklord-resourced'
]


DISTROS.each { |dist|
  PACKAGES.each { |pkg| 
    processPackage(dist, pkg)
  }
}

