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


type noopFilter string

// Load() method
func (p noopFilter) Init() error {
   return nil
}

func (p noopFilter) Process() error {
   return nil
}

func (p noopFilter) SigUsr1() error {
   return nil
}

func (p noopFilter) SigUsr2() error {
   return nil
}


// Exported symbol which supports filter.go's FilterAction type
var FilterPlugin noopFilter

