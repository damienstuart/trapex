#!/bin/bash

# Each plugin, by Golang 'plugin' library convention lives 
# in its own "package main" file.
# Practically speaking, this means that each plugin module code has to live
# in its own directory, or Golang tools get confused about which 'main'
# they need to be in.
#
# Each plugin type is separated out with its own directory structure,
# so we need to execute 'go build' in each directory.

# ---  functions  --------------------------------

function build_plugins() {
    ptype=$1
    echo "Building $ptype plugins:"
    for plugin in `ls -1 $ptype | egrep -v '\.so|\.go'`; do
        echo " - $ptype plugin: $plugin"
        (cd $ptype/$plugin && go build -buildmode=plugin -o ../$plugin.so $plugin.go)
    done
}

# ---  main  --------------------------------

targets=${1:-actions generators metrics}
for plugin_type in $targets ; do
    build_plugins $plugin_type
done

