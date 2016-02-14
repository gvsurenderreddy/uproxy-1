package main

import (
	"log"
)

type proxy struct {
	url                              string
	working                          bool
	delay, requests, sumTime, failed int64
}

type (
	proxies []proxy
)

type checkTask struct {
	id int
	pl *proxies
}

var (
	proxyListWorking proxies
)

func (pl proxies) Len() int {
	return len(pl)
}

func (pl proxies) Less(i, j int) bool {
	return pl[i].delay < pl[j].delay
}

func (pl proxies) Swap(i, j int) {
	pl[i], pl[j] = pl[j], pl[i]
}

func (pl *proxies) Add(url string, working bool, delay int64) {
	var p proxy
	p.url = url
	p.working = working
	p.delay = delay
	*pl = append(*pl, p)
}

func (pl proxies) Working() proxies {
	var plw proxies
	for i := range pl {
		if pl[i].working {
			plw = append(plw, pl[i])
		}
	}
	if len(plw) != 0 {
		return plw
	}
	return nil
}

func (pl *proxies) Check() {
	var ct checkTask
	ct.pl = pl
	pool := NewPool(cfg.WorkersCount)
	for i := range *pl {
		ct.id = i
		pool.Exec(ct)
	}

	pool.Close()
	pool.Wait()
	log.Printf("proxy check finished, found %v working proxies",
		len(pl.Working()))
	proxyListWorking = pl.Working()
}

func (c checkTask) Execute() {
	err := checkProxy(&(*c.pl)[c.id])
	(*c.pl)[c.id].requests++

	if err != nil {
		log.Printf("proxy %v not working: %v", (*c.pl)[c.id].url, err)
		(*c.pl)[c.id].working = false
		(*c.pl)[c.id].failed++
	} else {
		log.Printf("proxy %v working, delay %vms",
			(*c.pl)[c.id].url,
			(*c.pl)[c.id].delay)

		(*c.pl)[c.id].working = true
	}
}
