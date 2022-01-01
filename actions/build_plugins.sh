#!/bin/bash

echo "Building plugins:"
for source in `ls -1 plugins/*.go`; do
    plugin=`echo $source | sed -e 's/\.go$//'`
    echo " - plugin: $plugin"
    go build -buildmode=plugin -o $plugin.so $plugin.go
done

