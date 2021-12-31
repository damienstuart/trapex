#!/bin/bash

for source in `ls -1 plugins/*.go`; do
    plugin=`echo $source | sed -e 's/\.go$//'`
    go build -buildmode=plugin -o $plugin.so $plugin.go
done

