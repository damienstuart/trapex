// Copyright (c) 2021 Damien Stuart. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

/*
This plugin performs no useful action, but can be used for unit testing or performance
testing purposes.
 */

import (
  "fmt"
)

type filter_data struct {
    isLoaded bool
}

func (f filter_data) Init() error {
   fmt.Println("Load")
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
