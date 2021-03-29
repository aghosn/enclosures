#!/bin/sh

NAME="gosb"

#Cleanup.
printf "%`tput cols`s"|tr ' ' '.'
echo "Cleaning up"
#rm -f bin/*
rm -f /usr/local/bin/$NAME 2>/dev/null
