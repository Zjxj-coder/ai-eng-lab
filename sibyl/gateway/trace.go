package main

import (
	"encoding/json"
)

// Trace represents the lineage of a data point
type Trace struct {
	SQL       string `json:"sql"`
	MetricDef string `json:"metric_def"`
}

// DataPoint wraps a value with its trace
type DataPoint struct {
	Value interface{} `json:"value"`
	Trace Trace       `json:"trace"`
}

func NewDataPoint(value interface{}, sql, def string) DataPoint {
	return DataPoint{
		Value: value,
		Trace: Trace{
			SQL:       sql,
			MetricDef: def,
		},
	}
}

func (d *DataPoint) MarshalJSON() ([]byte, error) {
	type Alias DataPoint
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	})
}
