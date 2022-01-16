// Copyright (c) 2022 Kells Kearney. All rights reserved.
//
// Use of this source code is governed by the MIT License that can be found
// in the LICENSE file.
//
package pluginMeta

/*
 * Base interface for metrics, that allows programs to manage metrics in a
 * similar function
 */

// MetricDef defines the help text for the metrics
type MetricDef struct {
	Name string
	Help string
}

// String helper to return the name of the metric
func (m MetricDef) String() string {
	return m.Name
}

func CreateMetricDefs() []MetricDef {

	mymetrics := []MetricDef{
		MetricDef{Name: "incoming_traps_total",
			Help: "The total number of incoming SNMP traps",
		},
		MetricDef{Name: "handled_traps_total",
			Help: "The total number of handled SNMP traps",
		},
		MetricDef{Name: "dropped_traps_total",
			Help: "The total number of dropped SNMP traps",
		},
		MetricDef{Name: "ignored_traps_total",
			Help: "The total number of ignored SNMP traps",
		},
		MetricDef{Name: "v1_traps_total",
			Help: "The total number of SNMPv1 traps received",
		},
		MetricDef{Name: "v2c_traps_total",
			Help: "The total number of SNMPv2c traps received",
		},
		MetricDef{Name: "v3_traps_total",
			Help: "The total number of SNMPv3 traps received",
		},
	}

	return mymetrics
}
