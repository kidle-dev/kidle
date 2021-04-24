#!/usr/bin/env bash

target=$1
WHAT=$2
shift 2

IFS=","

for w in $WHAT
do
  if  [[ ! -d cmd/$w ]]
  then
    echo "cmd/$w does not exists"
  else
    (
      cd cmd/$w
      make $target $@
    )
  fi
done
