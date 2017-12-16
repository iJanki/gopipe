package proc

import (
    "crypto/md5"
    . "gopipe/core"
    log "github.com/sirupsen/logrus"
)

func init() {
    log.Info("Registering Md5Proc")
    GetRegistryInstance()["Md5Proc"] = NewMd5Proc
}

type Md5Proc struct {
    *ComponentBase
    InFields []string
    OutFields []string
    Salt string
}

func NewMd5Proc(inQ chan *Event, outQ chan *Event, cfg Config) Component {
    log.Info("Creating Md5Proc")

    in_fields := []string{}
    if tmp, ok := cfg["in_fields"].([]interface{}); ok {
        in_fields = InterfaceToStringArray(tmp)
    }

    out_fields := []string{}
    if tmp, ok := cfg["out_fields"].([]interface{}); ok {
        out_fields = InterfaceToStringArray(tmp)
    }

    log.Error(out_fields)

    salt, ok := cfg["salt"].(string)
    if !ok {
        salt = ""
    }

    m := &Md5Proc{NewComponentBase(inQ, outQ, cfg), in_fields, out_fields, salt}
    m.Tag = "MD5-LOG"
    return m
}


func (p *Md5Proc) Run() {
    log.Debug("Md5Proc Starting ... ")
    p.MustStop = false
    hash := md5.New()
    for !p.MustStop {
        e := <- p.InQ

        for i, ifield := range p.InFields {
            b, ok := e.Data[ifield].(string)
            if !ok {
                log.Error("Failed to convert field ", ifield, " to string...")
                continue
            }

            e.Data[p.OutFields[i]] = hash.Sum([]byte(b+p.Salt))
        }

        p.OutQ<-e

        // Stats
        p.StatsAddMesg()
        p.PrintStats()

    }

    log.Info("Md5Proc Stopping!?")
}
