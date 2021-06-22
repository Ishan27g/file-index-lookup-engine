#!/bin/sh 
# runs from root
[ -d "sample_files" ] && rm -rf sample_files
mkdir -p sample_files
CMD="./main -instance=$1 $2"
echo $CMD
$CMD
