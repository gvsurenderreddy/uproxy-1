#!/usr/bin/env bash

BINARIES="go dpkg-deb"

for binary in $BINARIES
do
type $binary >/dev/null 2>&1 || { echo >&2 "I require $binary but it's not installed.  Aborting."; exit 1; }
done

rm -rf *.deb
go get
GOOS=linux GOARCH=amd64 go build -o ./debian/usr/bin/uproxy

dpkg-deb --build debian 
mv debian.deb uproxy-0.0.1-1.deb
