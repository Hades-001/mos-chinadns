//     Copyright (C) 2020, IrineSistiana
//
//     This file is part of mos-chinadns.
//
//     mos-chinadns is free software: you can redistribute it and/or modify
//     it under the terms of the GNU General Public License as published by
//     the Free Software Foundation, either version 3 of the License, or
//     (at your option) any later version.
//
//     mos-chinadns is distributed in the hope that it will be useful,
//     but WITHOUT ANY WARRANTY; without even the implied warranty of
//     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//     GNU General Public License for more details.
//
//     You should have received a copy of the GNU General Public License
//     along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/IrineSistiana/mos-chinadns/dispatcher/config"
	"github.com/IrineSistiana/mos-chinadns/dispatcher/domainlist"
	"github.com/IrineSistiana/mos-chinadns/dispatcher/logger"
	"github.com/IrineSistiana/mos-chinadns/dispatcher/netlist"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/miekg/dns"

	"github.com/IrineSistiana/mos-chinadns/dispatcher"

	//DEBUG ONLY
	//_ "net/http/pprof"
	//"net/http"

	"github.com/sirupsen/logrus"
)

var (
	version = "dev/unknown"

	configPath  = flag.String("c", "config.yaml", "[path] load config from file")
	genConfigTo = flag.String("gen", "", "[path] generate a config template here")

	dir                 = flag.String("dir", "", "[path] change working directory to here")
	dirFollowExecutable = flag.Bool("dir2exe", false, "change working directory to the executable that started the current process")

	debug = flag.Bool("debug", false, "more log")
	quiet = flag.Bool("quiet", false, "no log")

	cpu         = flag.Int("cpu", runtime.NumCPU(), "the maximum number of CPUs that can be executing simultaneously")
	showVersion = flag.Bool("v", false, "show version info")

	probeDoTTimeout = flag.String("probe-dot-timeout", "", "[ip:port] probe dot server's idle timeout")
	probeTCPTimeout = flag.String("probe-tcp-timeout", "", "[ip:port] probe tcp server's idle timeout")

	benchIPListFile     = flag.String("bench-ip-list", "", "[path] benchmark ip search using this file")
	benchDomainListFile = flag.String("bench-domain-list", "", "[path] benchmark domain search using this file")

//DEBUG ONLY
//pprofAddr = flag.String("pprof", "", "[ip:port] DEBUG ONLY, hook http/pprof at this address")
)

func main() {

	//wait for signals
	go func() {
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)
		s := <-osSignals
		logrus.Infof("main: received signal: %v, program exited", s)
		os.Exit(0)
	}()

	flag.Parse()
	runtime.GOMAXPROCS(*cpu)

	//DEBUG ONLY
	//if len(*pprofAddr) != 0 {
	//	go func() {
	//		if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
	//			entry.Fatal("pprof backend is exited: %v", err)
	//		}
	//	}()
	//}

	// helper function

	// show version
	if *showVersion {
		fmt.Printf("%s\n", version)
		os.Exit(0)
	}

	// idle timeout test
	if len(*probeDoTTimeout) != 0 {
		err := probTCPTimeout(*probeDoTTimeout, true)
		if err != nil {
			logrus.Errorf("failed to prob server tcp idle connection timeout: %v", err)
		}
		os.Exit(0)
	}
	if len(*probeTCPTimeout) != 0 {
		err := probTCPTimeout(*probeTCPTimeout, false)
		if err != nil {
			logrus.Errorf("failed to prob server tls idle connection timeout: %v", err)
		}
		os.Exit(0)
	}

	// bench
	if len(*benchIPListFile) != 0 {
		err := benchIPList(*benchIPListFile)
		if err != nil {
			logrus.Errorf("bench ip list failed, %v", err)
		}
		os.Exit(0)
	}
	if len(*benchDomainListFile) != 0 {
		err := benchDomainList(*benchDomainListFile)
		if err != nil {
			logrus.Errorf("bench domain list failed, %v", err)
		}
		os.Exit(0)
	}

	// generate config
	if len(*genConfigTo) != 0 {
		err := config.GenConfig(*genConfigTo)
		if err != nil {
			logrus.Fatalf("main: can not generate config template, %v", err)
		} else {
			logrus.Info("main: config template generated")
		}
		os.Exit(0)
	}

	// main program starts here

	// show summary
	logrus.Infof("main: mos-chinadns ver: %s", version)
	logrus.Infof("main: arch: %s os: %s", runtime.GOARCH, runtime.GOOS)

	// try to change working dir to os.Executable() or *dir
	var wd string
	if *dirFollowExecutable {
		ex, err := os.Executable()
		if err != nil {
			logrus.Fatalf("main: failed to get executable path: %v", err)
		}
		wd = filepath.Dir(ex)
	} else {
		if len(*dir) != 0 {
			wd = *dir
		}
	}
	if len(wd) != 0 {
		err := os.Chdir(wd)
		if err != nil {
			logrus.Fatalf("main: failed to change the current working directory: %v", err)
		}
		logrus.Infof("main: current working directory: %s", wd)
	}

	//checking
	if len(*configPath) == 0 {
		logrus.Fatal("main: need a config file")
	}

	c, err := config.LoadConfig(*configPath)
	if err != nil {
		logrus.Fatalf("main: can not load config file, %v", err)
	}

	d, err := dispatcher.InitDispatcher(c)
	if err != nil {
		logrus.Fatalf("main: failed to init dispatcher: %v", err)
	}

	switch {
	case *quiet:
		logger.GetStd().SetLevel(logrus.ErrorLevel)
	case *debug:
		logger.GetStd().SetLevel(logrus.DebugLevel)
		go printStatus(time.Second * 10)
	default:
		logger.GetStd().SetLevel(logrus.InfoLevel)
	}

	err = d.StartServer()
	if err != nil {
		logrus.Fatalf("main: server exited with err: %v", err)
	}
}

func printStatus(d time.Duration) {
	m := new(runtime.MemStats)
	for {
		time.Sleep(d)
		runtime.ReadMemStats(m)
		logrus.Infof("printStatus: HeapObjects: %d NumGC: %d PauseTotalNs: %d, NumGoroutine: %d", m.HeapObjects, m.NumGC, m.PauseTotalNs, runtime.NumGoroutine())
	}
}

func probTCPTimeout(addr string, isTLS bool) error {
	q := new(dns.Msg)
	q.SetQuestion("www.google.com.", dns.TypeA)

	var conn net.Conn
	var err error

	logrus.Infof("connecting to %s", addr)
	if isTLS {
		tlsConfig := new(tls.Config)
		tlsConfig.InsecureSkipVerify = true
		tlsConn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dail tsl connection: %v", err)
		}
		tlsConn.SetDeadline(time.Now().Add(time.Second * 5))
		logrus.Info("connected, start TLS handshaking")
		err = tlsConn.Handshake()
		if err != nil {
			return fmt.Errorf("tls handshake failed: %v", err)
		}
		logrus.Info("TLS handshake completed")
		conn = tlsConn
	} else {
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("can not connect to server: %v", err)
		}
	}
	defer conn.Close()

	logrus.Info("sending request")
	conn.SetDeadline(time.Now().Add(time.Second * 3))
	dc := dns.Conn{Conn: conn}
	err = dc.WriteMsg(q)
	if err != nil {
		return fmt.Errorf("failed to write probe msg: %v", err)
	}
	logrus.Info("request sent, waiting for response")
	_, err = dc.ReadMsg()
	if err != nil {
		return fmt.Errorf("failed to read probe msg response: %v", err)
	}
	logrus.Info("response received")
	logrus.Info("waiting for peer to close the connection...")
	logrus.Info("this may take a while...")
	logrus.Info("if you think its long enough, to cancel the test, press Ctrl + C")
	conn.SetDeadline(time.Now().Add(time.Minute * 60))

	start := time.Now()
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err == nil {
		return fmt.Errorf("recieved unexpected data from peer: %v", buf[:n])
	}

	logrus.Infof("connection closed by peer after %.2f", time.Since(start).Seconds())
	return nil
}

func benchIPList(f string) error {
	list, err := netlist.NewListFromFile(f, true)
	if err != nil {
		return err
	}

	ipv6 := netlist.Conv(net.IPv4(8, 8, 8, 8).To16())

	start := time.Now()

	var n int = 1e6

	for i := 0; i < n; i++ {
		list.Contains(ipv6)
	}
	timeCost := time.Since(start)

	logrus.Infof("%d ns/op", timeCost.Nanoseconds()/int64(n))
	return nil
}

func benchDomainList(f string) error {
	list, err := domainlist.NewListFormFile(f, true)
	if err != nil {
		return err
	}
	start := time.Now()

	var n int = 1e6

	for i := 0; i < n; i++ {
		list.Has("com.")
	}
	timeCost := time.Since(start)

	logrus.Infof("%d ns/op", timeCost.Nanoseconds()/int64(n))
	return nil
}
