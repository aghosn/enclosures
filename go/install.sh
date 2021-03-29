#!/bin/sh
NAME="gosb"
CURRENT=`pwd`

FILE="gosb_run.sh"

#Creating the command file.
if [ -f "$FILE" ]; then
	rm -f $FILE
fi

echo "
#!/bin/bash
CURRENT=\"$CURRENT\"
GOROOT=$CURRENT $CURRENT/bin/go \$@
" > $FILE

chmod +x $FILE

# Installing command.
printf "%`tput cols`s"|tr ' ' '.'
echo "Installing cmd as $NAME"
if [ -f "/usr/local/bin/$NAME" ]; then
  rm /usr/local/bin/$NAME
fi
ln -s $CURRENT/$FILE /usr/local/bin/$NAME
echo "Intalled as: `which $NAME`"
