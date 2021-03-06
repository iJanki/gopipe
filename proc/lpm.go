/*
   - LPM: Longest Prefix Match: Loads a list of prefixes (network addresses)
   periodically from a file and performs LPM for event data fields. The result
   can have meta-data which are then exported back to the event's data. For
   example the following will figure out the matched prefix and Autonomous
   System number (ASN) and attach then to the event's data:

       {
           "module": "LPMProc",
           "filepath": "/tmp/prefix-asn.txt",
           "reload_minutes": 1440,
           "in_fields": ["src", "dst"],
           "out_fields": [
               {"newkey": "_{{in_field}}_prefix", "metakey": "prefix"},
               {"newkey": "_{{in_field}}_asn", "metakey": "asn"}
           ]
       }

       // Input data are in the format

           prefix/len JSON-metadata

       Example:

           10.0.0.0/8 {"asn": -1, "owner": "me"}

       Output:

           {..."_src_asn": -1, "_dst_asn": -1, "_src_prefix": "10.0.0.0/8" ...}
*/
package proc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	// "github.com/asergeyev/nradix"
	"github.com/b1v1r/nradix"
	log "github.com/sirupsen/logrus"
	"github.com/urban-1/gopipe/core"
)

func init() {
	log.Info("Registering LPMProc")
	core.GetRegistryInstance()["LPMProc"] = NewLPMProc
}

type LPMOutField struct {
	NewKey  string
	MetaKey string
}

type LPMProc struct {
	*core.ComponentBase
	Tree          *nradix.Tree
	TreeLock      *sync.Mutex
	FilePath      string
	ReloadMinutes int
	InFields      []string
	OutFields     []LPMOutField
}

func NewLPMProc(inQ chan *core.Event, outQ chan *core.Event, cfg core.Config) core.Component {
	log.Info("Creating LPMProc")

	fpath, ok := cfg["filepath"].(string)
	if !ok {
		panic("LPMProc: no valid filepath supplied. This is required...")
	}

	in_fields := []string{}
	if tmp, ok := cfg["in_fields"].([]interface{}); ok {
		in_fields = core.InterfaceToStringArray(tmp)
	}

	out_fields := []LPMOutField{}
	tmpof, ok := cfg["out_fields"].([]interface{})
	for _, v := range tmpof {
		v2 := v.(map[string]interface{})
		out_fields = append(
			out_fields,
			LPMOutField{v2["newkey"].(string), v2["metakey"].(string)})
	}

	m := &LPMProc{core.NewComponentBase(inQ, outQ, cfg),
		nradix.NewTree(100), &sync.Mutex{}, fpath,
		int(cfg["reload_minutes"].(float64)),
		in_fields, out_fields}

	m.Tag = "PROC-LPM"
	return m
}

func (p *LPMProc) Signal(signal string) {
	log.Infof("LPMProc Received signal '%s'", signal)
	switch signal {
	case "reload":
		p.loadTree()
	default:
		log.Infof("LPMProc UNKNOW signal '%s'", signal)
	}
}

func (p *LPMProc) Run() {
	log.Debug("LPMProc Starting ... ")

	// Spawn the loader
	if p.ReloadMinutes > 0 {
		// Periodic reloading
		go func(p *LPMProc) {
			p.loadTree()
			time.Sleep(time.Duration(p.ReloadMinutes) * time.Minute)
		}(p)
	} else {
		// Once off
		p.loadTree()
	}

	p.MustStop = false
	cfg_error := false

	for !p.MustStop {
		// Do not read until we lock the tree!
		log.Debug("LPMProc Reading stop=", p.MustStop)
		e, err := p.ShouldRun()
		if err != nil {
			continue
		}

		p.TreeLock.Lock()

		for _, ifield := range p.InFields {

			what, ok := e.Data[ifield].(string)
			if !ok {
				// This is a user error, maybe error once?
				if !cfg_error {
					log.Error("Cannot find field ", ifield)
					cfg_error = true
				}
				continue
			}
			// Get the node
			meta, err := p.Tree.FindCIDR(what)
			if err != nil {
				log.Error("LPM error in find: ", err.Error())
				continue
			}

			// Generate new fields
			for _, ofield := range p.OutFields {
				new_field := strings.Replace(ofield.NewKey, "{{in_field}}", ifield, 1)
				if meta == nil {
					log.Debug("Could not find prefix for '", ifield, "' -> ", e.Data[ifield].(string))
					e.Data[new_field] = ""
				} else {
					e.Data[new_field] = meta.(map[string]interface{})[ofield.MetaKey]
				}
			}
		}

		// Now unlock and push
		p.TreeLock.Unlock()

		p.OutQ <- e

		// Stats
		p.StatsAddMesg()
		p.PrintStats()

	}

	log.Info("LPMProc Stopping")
}

func (p *LPMProc) loadTree() {

	f, err := os.Open(p.FilePath)
	if err != nil {
		log.Error("LPM: Could not load prefix file")
		return
	}
	p.TreeLock.Lock()

	p.Tree = nradix.NewTree(0)

	log.Warn("LPM: Reading file")
	reader := bufio.NewReader(f)

	count := 1

	line, _, err := reader.ReadLine()
	for err != io.EOF {
		// Skip comments
		if len(line) > 0 && string(line[0]) == "#" {
			line, _, err = reader.ReadLine()
			continue
		}

		json_data := map[string]interface{}{}
		parts := bytes.SplitAfterN(line, []byte(" "), 2)
		parts[0] = bytes.Trim(parts[0], " ")
		meta := bytes.Join(parts[1:], []byte(""))

		d := json.NewDecoder(bytes.NewReader(meta))
		d.UseNumber()

		if d.Decode(&json_data) != nil {
			log.Error("LPM: Unable to parse prefix meta-data: ", string(meta))
			line, _, err = reader.ReadLine()
			continue
		}

		json_data["prefix"] = string(parts[0])
		err = p.Tree.AddCIDRb(parts[0], json_data)
		if err != nil {
			log.Error(err)
		} else {
			count += 1
		}
		line, _, err = reader.ReadLine()
	}

	log.Info("LPM: Done! Loaded ", count, " prefixes!")
	f.Close()
	p.TreeLock.Unlock()

}
