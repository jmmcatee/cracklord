#!/usr/bin/env bash

VER=`git describe --dirty --always --tags`
BASEDIR="."
QSRC="$BASEDIR/build/queued/"
RSRC="$BASEDIR/build/resourced/"

BUILDDIR="./cracklord_build"
QDST="$BUILDDIR/queue"
RDST="$BUILDDIR/resource"

WEBDIR="./web"
URL="http://jmmcatee.github.io/cracklord/"
MAINT="jmmcatee@gmail.com"

echo -n "Creating queue server package directories..."
mkdir -p $QDST/usr/bin
go get -v ./cmd/queued
go build -v -o $QDST/usr/bin/cracklord-queued ./cmd/queued
mkdir -p $QDST/etc/cracklord
cp -r $QSRC/conf/* $QDST/etc/cracklord/
mkdir -p $QDST/var/cracklord/www
cp -r $BASEDIR/web/* $QDST/var/cracklord/www
mkdir -p $QDST/etc/init
cp -r $QSRC/cracklord-queued.conf $QDST/etc/init/
echo "done"

echo -n "Creating resource server package directories"
mkdir -p $RDST/usr/bin
go get -v ./cmd/resourced
go build -v -o $RDST/usr/bin/cracklord-resourced ./cmd/resourced
mkdir -p $RDST/etc/cracklord
cp -r $RSRC/conf/* $RDST/etc/cracklord/
mkdir -p $RDST/etc/init
cp -r $RSRC/cracklord-resourced.conf $RDST/etc/init/
echo "done"

echo -n "Generating queue package using FPM"
fpm -s dir -t deb -n "cracklord-queued" -v "$VER" --before-install $QSRC/beforeinstall.sh --after-install $QSRC/afterinstall.sh --before-remove $QSRC/beforeremove.sh --after-remove $QSRC/afterremove.sh --url "$URL" --config-files etc/cracklord --description "CrackLord job management queue server" -m "$MAINT" -C $QDST usr etc var
echo "done"

echo -n "Generating resource package using FPM"
fpm -s dir -t deb -n "cracklord-resourced" -v "$VER" --before-install $RSRC/beforeinstall.sh --after-install $RSRC/afterinstall.sh --before-remove $RSRC/beforeremove.sh --after-remove $RSRC/afterremove.sh --url "$URL" --config-files etc/cracklord --description "Cracklord job management system resource server" -m "$MAINT" -C $RDST usr etc
echo "done"