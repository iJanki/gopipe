package proc

import (
	. "github.com/urban-1/gopipe/core"
	. "github.com/urban-1/gopipe/tests"
	"fmt"
	"testing"
	"runtime"
	"path"
)

func getMockFile() string {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("Could not get caller")
	}
	filepath := path.Join(path.Dir(filename), "../tests/prefix-asn1")
	return filepath
}

func getLPM(in chan *Event, out chan *Event) Component {
	strcfg := fmt.Sprintf(`{
		"filepath": "%s",
		"reload_minutes": 1440,
		"in_fields": ["src", "dst"],
		"out_fields": [
			{"newkey": "_{{in_field}}_prefix", "metakey": "prefix"},
			{"newkey": "_{{in_field}}_asn", "metakey": "asn"},
			{"newkey": "_{{in_field}}_comment", "metakey": "comment"}
		]}`, getMockFile())

	comp := NewLPMProc(in, out, GetConfig(strcfg))
	comp.Signal("reload")
	return comp
}

func TestLPMV4Outer(t *testing.T) {
	in, out := GetChannels()
	in <- GetEvent(`{"src": "176.52.166.10"}`)

	comp := getLPM(in, out)
	go comp.Run()

	e := <-out
	if _, ok := e.Data["_src_prefix"]; !ok {
		t.Error("LPM did not add anything!!")
	}
	comment, ok := e.Data["_src_comment"];
	if !ok {
		t.Error("LPM Did not add field comment...")
	}
	if comment != "v4 outer-range" {
		t.Error("Failed to match the correct prefix:")
		t.Error(e.Data)
	}
}

func TestLPMV4Inner(t *testing.T) {
	in, out := GetChannels()
	in <- GetEvent(`{"src": "176.52.166.195"}`)

	comp := getLPM(in, out)
	go comp.Run()

	e := <-out
	if _, ok := e.Data["_src_prefix"]; !ok {
		t.Error("LPM did not add anything!!")
	}
	comment, ok := e.Data["_src_comment"];
	if !ok {
		t.Error("LPM Did not add field comment...")
	}
	if comment != "v4 inner-range" {
		t.Error("Failed to match the correct prefix:")
		t.Error(e.Data)
	}
}

// func TestLPMShouldNotRun(t *testing.T) {
// 	in, out := GetChannels()
// 	in <- GetEventRun(`{"doesnt": "matter"}`, false)
//
// 	comp := getLPM()
// 	go comp.Run()
//
// 	e := <-out
// 	if _, ok := e.Data["test"]; ok {
// 		// Has the new value!!! raise error
// 		t.Error("LPM: run when it shouldnt...")
// 		t.Error(e.Data)
// 	}
// }
//
// func TestLPMShouldRun(t *testing.T) {
// 	in, out := GetChannels()
// 	in <- GetEventRun(`{"doesnt": "matter"}`, true)
//
// 	comp := NewLPMProc(in, out, GetConfig(`{"field_name":"ts"}`))
// 	go comp.Run()
//
// 	e := <-out
// 	if _, ok := e.Data["test"]; ok {
// 		// Has the new value!!! raise error
// 		t.Error("LPM: didn't run when it should...")
// 		t.Error(e.Data)
// 	}
// }
