VER=`git describe --dirty --always --tags`

mkdir -p $QUEUEDIR/usr/bin
go get -v ./cmd/queued
go build -v -o $QUEUEDIR/usr/bin/cracklord-queued ./cmd/queued
mkdir -p $QUEUEDIR/etc/cracklord
cp -r $QUEUESRC/conf/* $QUEUEDIR/etc/cracklord/
mkdir -p $QUEUEDIR/var/cracklord/www
cp -r web $QUEUEDIR/var/cracklord/www
mkdir -p $QUEUEDIR/etc/init
cp -r $QUEUESRC/cracklord-queued.conf $QUEUEDIR/etc/init/

mkdir -p $RESOURCEDIR/usr/bin
go get -v ./cmd/resourced
go build -v -o $RESOURCEDIR/usr/bin/cracklord-resourced ./cmd/resourced
mkdir -p $RESOURCEDIR/etc/cracklord
cp -r $RESOURCESRC/conf/* $RESOURCEDIR/etc/cracklord/
mkdir -p $RESOURCEDIR/etc/init
cp -r $RESOURCESRC/cracklord-resourced.conf $RESOURCEDIR/etc/init/

gem install fpm --quiet
gem install package_cloud --quiet
FPM=`rbenv which fpm`
PC=`rbenv which package_cloud`

$FPM -s dir -t deb -n "cracklord-queued" -v "$VER" --before-install $QUEUESRC/beforeinstall.sh --after-install $QUEUESRC/afterinstall.sh --before-remove $QUEUESRC/beforeremove.sh --after-remove $QUEUESRC/afterremove.sh --url "$URL" --config-files etc/cracklord --description "CrackLord job management queue server" -m "$MAINT" -C $QUEUEDIR usr etc
$FPM -s dir -t deb -n "cracklord-resourced" -v "$VER" --before-install $RESOURCESRC/beforeinstall.sh --after-install $RESOURCESRC/afterinstall.sh --before-remove $RESOURCESRC/beforeremove.sh --after-remove $RESOURCESRC/afterremove.sh --url "$URL" --config-files etc/cracklord --description "Cracklord job management system resource server" -m "$MAINT" -C $RESOURCEDIR usr etc

$PC push emperorcow/cracklord/debian/jessie *.deb
$PC push emperorcow/cracklord/ubuntu/trusty *.deb