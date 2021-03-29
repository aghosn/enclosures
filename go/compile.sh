#!/bin/sh

NAME="gosb"
CURRENT=`pwd`

if test -z "$GOSB_BOOT"; then
  echo "You need to set GOSB_BOOT to the extracted gosb_bootstrap.tar.gz folder"
  exit 1
fi

# Compiling
printf "%`tput cols`s"|tr ' ' '.'
echo "Compiling"
cd src/
GOROOT_BOOTSTRAP=$GOSB_BOOT GOOS=linux GOARCH=amd64 ./make.bash --no-banner
cd ..
