/*
   - UDP: Send UDP datagrams out ... Particularly useful for flow sampler and
   replication configurations
*/
package output

import (
	"encoding/json"
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/urban-1/gopipe/core"
)

func init() {
	log.Info("Registering UDPJSONOutput")
	core.GetRegistryInstance()["UDPJSONOutput"] = NewUDPJSONOutput

	log.Info("Registering UDPCSVOutput")
	core.GetRegistryInstance()["UDPCSVOutput"] = NewUDPStrOutput

	log.Info("Registering UDPRawOutput")
	core.GetRegistryInstance()["UDPRawOutput"] = NewUDPRawOutput

	log.Info("Registering UDPStrOutput")
	core.GetRegistryInstance()["UDPStrOutput"] = NewUDPStrOutput
}

// The base structure for common UDP Ops
type UDPJSONOutput struct {
	*core.ComponentBase
	// Keep a referece to the struct responsible for decoding...
	Encoder core.LineCodec
	target  string
	port    uint32
	Sock    net.Conn
}

func NewUDPJSONOutput(inQ chan *core.Event, outQ chan *core.Event, cfg core.Config) core.Component {
	log.Info("Creating UDPJSONOutput")
	m := UDPJSONOutput{core.NewComponentBase(inQ, outQ, cfg),
		&core.JSONLineCodec{},
		cfg["target"].(string), uint32(cfg["port"].(float64)), nil}

	m.Tag = "OUT-UDP-JSON"

	return &m
}

func (p *UDPJSONOutput) Signal(string) {}

func (p *UDPJSONOutput) Run() {
	pstr := strconv.FormatInt(int64(p.port), 10)

	//Connect udp
	conn, err := net.Dial("udp", p.target+":"+pstr)
	if err != nil {
		log.Error("UDP-OUT: Failed to connect: ", err.Error())
		return
	}
	defer conn.Close()

	// Avoid alloc in loops
	var data []byte

	for {
		e, err := p.ShouldRun()
		if err != nil {
			continue
		}

		data, err = p.Encoder.ToBytes(e.Data)
		if err != nil {
			log.Error("UDP-OUT: Failed to encode data: ", err.Error())
			continue
		}

		//simple write
		conn.Write(data)

		// Check if we are being used in proc!
		if p.OutQ != nil {
			p.OutQ <- e
		}

		// Stats
		p.StatsAddMesg()
		p.PrintStats()
	}

}

// UDP CSV Implementation
type UDPCSVOutput struct {
	*UDPJSONOutput
}

func NewUDPCSVOutput(inQ chan *core.Event, outQ chan *core.Event, cfg core.Config) core.Component {
	log.Info("Creating UDPCSVOutput")

	// Defaults...
	m := UDPCSVOutput{NewUDPJSONOutput(inQ, outQ, cfg).(*UDPJSONOutput)}

	m.Tag = "OUT-UDP-CSV"

	// Change to CSV
	c := &core.CSVLineCodec{Headers: nil, Separator: ","[0], Convert: true}
	cfgbytes, _ := json.Marshal(cfg)
	json.Unmarshal(cfgbytes, c)
	m.Encoder = c

	return &m
}

// UDP Raw Implementation
type UDPRawOutput struct {
	*UDPJSONOutput
}

func NewUDPRawOutput(inQ chan *core.Event, outQ chan *core.Event, cfg core.Config) core.Component {
	log.Info("Creating UDPRawOutput")

	// Defaults...
	m := UDPRawOutput{NewUDPJSONOutput(inQ, outQ, cfg).(*UDPJSONOutput)}

	m.Tag = "OUT-UDP-RAW"

	// Change to CSV
	m.Encoder = &core.RawLineCodec{}

	return &m
}

// UDP String implementation
type UDPStrOutput struct {
	*UDPJSONOutput
}

func NewUDPStrOutput(inQ chan *core.Event, outQ chan *core.Event, cfg core.Config) core.Component {
	log.Info("Creating UDPStrOutput")

	// Defaults...
	m := UDPStrOutput{NewUDPJSONOutput(inQ, outQ, cfg).(*UDPJSONOutput)}

	m.Tag = "OUT-UDP-STR"

	// Change to CSV
	m.Encoder = &core.StringLineCodec{}

	return &m
}
