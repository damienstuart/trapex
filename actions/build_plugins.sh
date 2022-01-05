#!/bin/bash

echo "Building plugins:"
for plugin in `ls -1 plugins`; do
    echo " - plugin: $plugin"
    (cd plugins/$plugin && go build -buildmode=plugin -o ../$plugin.so $plugin.go)
done

