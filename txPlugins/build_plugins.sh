#!/bin/bash

echo "Building plugins:"
for plugin in `ls -1 actions | grep -v .so`; do
    echo " - Filter action plugin: $plugin"
    (cd actions/$plugin && go build -buildmode=plugin -o ../$plugin.so $plugin.go)
done

