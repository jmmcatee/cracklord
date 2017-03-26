#!/usr/bin/env bash

VER=`git describe --dirty --always --tags`
BASEDIR="github.com/jmmcatee/cracklord"
QSRC="$BASEDIR/build/queued/"
RSRC="$BASEDIR/build/resourced/"

BUILDDIR="./cracklord_build"
QDST="$BUILDDIR/queue"
RDST="$BUILDDIR/resource"

WEBDIR="./web"
URL="http://jmmcatee.github.io/cracklord/"
MAINT="emperorcow@gmail.com"

echo "Creating queue server package directories"
mkdir -p $QDST/usr/bin 
mkdir -p $QDST/etc/cracklord
cp -r ub.com/conf/* $QDST/etc/cracklord/
mkdir -p $QDST/var/cracklord/www
cp -r web $QDST/var/cracklord/www
mkdir -p $QDST/etc/init
cp -r $QSRC/cracklord-queued.conf $QDST/etc/init/

echo "Creating resource server package directories"
mkdir -p $RDST/usr/bin
mkdir -p $RDST/etc/cracklord
cp -r $RSRC/conf/* $RDST/etc/cracklord/
mkdir -p $RDST/etc/init
cp -r $RSRC/cracklord-resourced.conf $RDST/etc/init/

echo -n "Generating packages using FPM"
FPM=`rbenv which fpm`
$FPM -s dir -t deb -n "cracklord-queued" -v "$VER" --before-install $QSRC/beforeinstall.sh --after-install $QSRC/afterinstall.sh --before-remove $QSRC/beforeremove.sh --after-remove $QSRC/afterremove.sh --url "$URL" --config-files etc/cracklord --description "CrackLord job management queue server" -m "$MAINT" -C $QSRC usr etc var
$FPM -s dir -t deb -n "cracklord-resourced" -v "$VER" --before-install $RESOURCESRC/beforeinstall.sh --after-install $RESOURCESRC/afterinstall.sh --before-remove $RESOURCESRC/beforeremove.sh --after-remove $RESOURCESRC/afterremove.sh --url "$URL" --config-files etc/cracklord --description "Cracklord job management system resource server" -m "$MAINT" -C $RDST usr etc

