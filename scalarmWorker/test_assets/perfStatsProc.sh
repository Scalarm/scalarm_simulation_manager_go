#!/bin/bash

if [ $1 -eq 1 ]
then

  x=5
  while :
  do
    x=$((x*5 / 3))
  done

else

  echo "Proc management"

  PID1="$!"

  sh $0 1 &

  sh $0 1 &

  wait 

fi
