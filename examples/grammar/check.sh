#!/bin/bash

DIR=${1%/}

if [[ -z "$DIR" ]] ; then
  echo "usage is  check.sh <sample_directory>"
  exit 1;
fi

check-file() { # file_name arg...
  name="$1"
  shift
  echo "$name"
  extra=""
  if [[ "$name" = *.txt ]] ; then
    extra="-m";
  fi
  go run grammar.go $extra $@ "$DIR/.grammar" "$name" >/dev/null
}

for name in $DIR/valid/* ; do
  check-file $name;
done

for name in $DIR/invalid/* ; do
  check-file $name -e;
done
