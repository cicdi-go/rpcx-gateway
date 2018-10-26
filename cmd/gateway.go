package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"crypto/tls"
	"github.com/docker/libkv/store"
	gateway "github.com/rpcx-ecosystem/rpcx-gateway"
	"github.com/smallnest/rpcx/client"
)

var (
	addr       = flag.String("addr", ":9982", "http server address")
	st         = flag.String("st", "http1", "server type: http1 or h2c")
	registry   = flag.String("registry", "etcd://172.16.12.228:2379,172.16.12.229:2379,172.16.12.230:2379", "registry address")
	basePath   = flag.String("basepath", "/rpcx_test", "basepath for zookeeper, etcd and consul")
	failmode   = flag.Int("failmode", int(client.Failover), "failMode, Failover in default")
	selectMode = flag.Int("selectmode", int(client.RoundRobin), "selectMode, RoundRobin in default")
	isTls      = flag.Bool("tls", true, "is tls mode")
	certFile   = flag.String("cert", "ssl/etcd.pem", "cert file path")
	keyFile    = flag.String("key", "ssl/etcd-key.pem", "cert key file path")
)

func main() {
	flag.Parse()

	d, err := createServiceDiscovery(*registry)
	if err != nil {
		log.Fatal(err)
	}
	gw := gateway.NewGateway(*addr, gateway.ServerType(*st), d, client.FailMode(*failmode), client.SelectMode(*selectMode), client.DefaultOption)

	gw.Serve()
}

func createServiceDiscovery(regAddr string) (client.ServiceDiscovery, error) {
	i := strings.Index(regAddr, "://")
	if i < 0 {
		return nil, errors.New("wrong format registry address. The right fotmat is [registry_type://address]")
	}

	regType := regAddr[:i]
	regAddr = regAddr[i+3:]
	var options *store.Config
	if *isTls {
		cer, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatal(err)
		}
		options = &store.Config{
			TLS: &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cer},
			},
		}
	}
	switch regType {
	case "peer2peer": //peer2peer://127.0.0.1:8972
		return client.NewPeer2PeerDiscovery("tcp@"+regAddr, ""), nil
	case "multiple":
		var pairs []*client.KVPair
		pp := strings.Split(regAddr, ",")
		for _, v := range pp {
			pairs = append(pairs, &client.KVPair{Key: v})
		}
		return client.NewMultipleServersDiscovery(pairs), nil
	case "zookeeper":
		return client.NewZookeeperDiscoveryTemplate(*basePath, strings.Split(regAddr, ","), options), nil
	case "etcd":
		return client.NewEtcdDiscoveryTemplate(*basePath, strings.Split(regAddr, ","), options), nil
	case "consul":
		return client.NewConsulDiscoveryTemplate(*basePath, strings.Split(regAddr, ","), options), nil
	case "mdns":
		client.NewMDNSDiscoveryTemplate(10*time.Second, 10*time.Second, "")
	default:
		return nil, fmt.Errorf("wrong registry type %s. only support peer2peer,multiple, zookeeper, etcd, consul and mdns", regType)
	}

	return nil, errors.New("wrong registry type. only support peer2peer,multiple, zookeeper, etcd, consul and mdns")
}
