// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
Dump data out in Clickhouse CSV data format, for adding to a Clickhouse database
 */

import (
  "fmt"
)

const plugin_name = "Clickhouse"

type filter_data struct {
    isLoaded bool
}

func (f filter_data) Init() error {
   fmt.Println("initialized " + plugin_name)
   return nil
}

func (f filter_data) Process() error {
   fmt.Println("Action")
   return nil
}

func (f filter_data) SigUsr1() error {
   fmt.Println("SigUsr1")
   return nil
}

func (f filter_data) SigUsr2() error {
   fmt.Println("SigUsr2")
   return nil
}


var FilterPlugin filter_data
