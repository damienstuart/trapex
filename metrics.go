// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package main

import (
	pluginMeta "github.com/damienstuart/trapex/txPlugins"
)

// These constants should be in the same order as the createMetricDefs function returns them
const (
	TrapCount int = iota
	HandledTraps
	DroppedTraps
	IgnoredTraps
	V1Traps
	V2cTraps
	V3Traps
)

func createMetricDefs() []pluginMeta.MetricDef {

	mymetrics := []pluginMeta.MetricDef{
		pluginMeta.MetricDef{Name: "incoming_traps_total",
			Help: "The total number of incoming SNMP traps",
		},
		pluginMeta.MetricDef{Name: "handled_traps_total",
			Help: "The total number of handled SNMP traps",
		},
		pluginMeta.MetricDef{Name: "dropped_traps_total",
			Help: "The total number of dropped SNMP traps",
		},
		pluginMeta.MetricDef{Name: "ignored_traps_total",
			Help: "The total number of ignored SNMP traps",
		},
		pluginMeta.MetricDef{Name: "v1_traps_total",
			Help: "The total number of SNMPv1 traps received",
		},
		pluginMeta.MetricDef{Name: "v2c_traps_total",
			Help: "The total number of SNMPv2c traps received",
		},
		pluginMeta.MetricDef{Name: "v3_traps_total",
			Help: "The total number of SNMPv3 traps received",
		},
	}

	return mymetrics
}
