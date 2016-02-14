package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	netTimeout      = 20 * time.Second
	defaultTimeout  = 5 * time.Second
	defaultInterval = 60 * time.Minute
)

var (
	timeout  = defaultTimeout
	interval = defaultInterval
)

//TODO: https
//TODO: auth

func checkProxy(p *proxy) (err error) {

	proxyURL, err := url.Parse("http://" + p.url)

	if err != nil {
		log.Printf("error parsing url: %v", p.url)
	}

	var startDuration = time.Now().UnixNano()
	proxiedClient := &http.Client{Transport: &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(timeout)
			c, err := net.DialTimeout(netw, addr, timeout)
			if err != nil {
				return nil, err
			}
			c.SetDeadline(deadline)
			return c, nil
		},
		Proxy: http.ProxyURL(proxyURL)}, Timeout: timeout}
	resp, err := proxiedClient.Get(cfg.Check.URL)
	var stopDuration = time.Now().UnixNano()
	if err != nil {
		return err
	}

	delay := (stopDuration - startDuration) / int64(time.Millisecond)

	if resp.StatusCode != 200 {
		return fmt.Errorf("proxy return code not 200: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error: %v", err)
		return err
	}

	if strings.Contains(string(body), cfg.Check.String) {
		p.delay = delay
		p.sumTime += delay
		return nil
	}

	return fmt.Errorf("response not contain \"%v\"", cfg.Check.String)
}

//TODO: gaussian distribution by avg response time
func (pl *proxies) getRandomProxy() (*proxy, error) {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	if len(*pl) == 0 {
		return &proxy{}, fmt.Errorf("working proxy not found")
	}
	return &(*pl)[r1.Intn(len(*pl))], nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func logAndSendError(w *http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	http.Error(*w, err.Error(), 502)
}

func request(w http.ResponseWriter, r *http.Request) {
	var (
		err  error
		resp *http.Response
		req  *http.Request
		p    *proxy
	)

	/*if r.Method == "CONNECT" {
		err = fmt.Errorf("https not supported, client: %v, URI: %v",
		 r.RemoteAddr,
		 req.URL.String())
		logAndSendError(&w, err)
		return
	}*/

	req, err = http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		logAndSendError(&w, err)
		return
	}
	req.Proto = "HTTP/1.0"
	req.ProtoMajor = 1
	req.ProtoMinor = 0
	//req.URL.Scheme = "http"
	req.Close = true
	req.RequestURI = ""
	/*
		r.Header.Del("Accept-Encoding")
		r.Header.Del("Proxy-Connection")
		r.Header.Del("Proxy-Authenticate")
		r.Header.Del("Proxy-Authorization")
		r.Header.Del("Connection")
		r.Header.Del("Content-Length")*/

	copyHeader(req.Header, r.Header)

	for i := 0; i < cfg.MaxTry; i++ {
		p, err = proxyListWorking.getRandomProxy()
		if err != nil {
			logAndSendError(&w, err)
			return
		}
		proxyURL, err := url.Parse("http://" + p.url)
		proxiedClient := &http.Client{Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				deadline := time.Now().Add(timeout)
				c, err := net.DialTimeout(netw, addr, timeout)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
			Proxy: http.ProxyURL(proxyURL)}}

		var startDuration = time.Now().UnixNano()
		resp, err = proxiedClient.Do(req)
		if err == nil {
			var stopDuration = time.Now().UnixNano()
			duration := (stopDuration - startDuration) / int64(time.Millisecond)
			if cfg.Debug {
				log.Printf("proxy: %v, client: %v, URI: %v, code: %v, time: %vms",
					p.url,
					r.RemoteAddr,
					req.URL.String(),
					resp.StatusCode,
					strconv.FormatInt(duration, 10))
			}
			copyHeader(w.Header(), resp.Header)

			if _, err = io.Copy(w, resp.Body); err != nil {
				logAndSendError(&w, fmt.Errorf("%v, proxy: %v, Method: %v, URI: %v",
					err,
					p.url,
					req.Method,
					req.URL.String()))
			} else {
				p.requests++
				p.sumTime += duration
				break
			}
		} else {
			p.requests++
			p.failed++
			p.working = false
			log.Printf("request failed, error: %v", err)
		}
	}
}

func setupSignals() {
	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)
	go func() {
		for {
			<-sigusr1
			sort.Sort(proxyListWorking)
			log.Printf("=============== list of working proxies, total %v ===============", len(proxyListWorking))
			for i := range proxyListWorking {
				if proxyListWorking[i].requests == 0 {
					proxyListWorking[i].requests = -1
				}
				log.Printf("proxy: %v, delay %vms, requests: %v, avg response time: %v, failed: %v",
					proxyListWorking[i].url,
					proxyListWorking[i].delay,
					proxyListWorking[i].requests,
					proxyListWorking[i].sumTime/proxyListWorking[i].requests,
					proxyListWorking[i].failed)

			}
		}
	}()
}

func status(w http.ResponseWriter, r *http.Request) {
	var requests int64
	for _, p := range proxyListWorking {
		requests += p.requests
	}
	fmt.Fprintf(w, "working:%v\nrequests:%v", len(proxyListWorking), requests)
}

func main() {
	var (
		proxyRegexp = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\:[0-9]{1,5}\b`)
		proxyList   proxies
	)

	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := readConfig(); err != nil {
		log.Fatalln("couldn't read config: %v", err)
	}

	if !strings.Contains(cfg.Check.URL, "http://") {
		cfg.Check.URL = "http://" + cfg.Check.URL
	}

	t, err := time.ParseDuration(cfg.Check.Timeout)
	if err != nil {
		log.Printf("unknown timeout format, set to default %v, error: %v",
			defaultTimeout,
			err)
	} else {
		timeout = t
		log.Printf("set check timeout to %v", t)
	}

	i, err := time.ParseDuration(cfg.Check.Interval)
	if err != nil {
		log.Printf("unknown interval format, set to default %v, error: %v",
			defaultInterval,
			err)
	} else {
		interval = i
		log.Printf("set check interval to %v", i)
	}

	proxyListFile, err := os.Open(cfg.ProxyList)
	defer proxyListFile.Close()
	if err != nil {
		log.Fatalf("couldn't open proxy list %v: %v", cfg.ProxyList, err)
	}

	r := bufio.NewReaderSize(proxyListFile, 256)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		s := string(line)
		if proxyRegexp.MatchString(s) {
			proxyList.Add(s, false, 0)
		} else {
			log.Printf("failed entry in proxy list: %v", s)
		}
		line, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		log.Print("error in reading proxy list: buffer size to small")
		return
	}
	if err != io.EOF {
		log.Printf("error in reading proxy list: %v", err)
		return
	}

	log.Print("begin initial proxy check")
	proxyList.Check()

	setupSignals()

	ticker := time.NewTicker(interval)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				proxyList.Check()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	http.HandleFunc("/", request)
	http.HandleFunc("/status", status)
	log.Print("started")
	log.Fatal(http.ListenAndServe(cfg.Bind, nil))
}
