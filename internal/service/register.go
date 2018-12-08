package service

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/spf13/viper"
	"log"
)

var scheme string

func Registration(addr string, port int, ssl bool)  {
	consul := viper.GetStringMapString("consul")
	check := viper.GetStringMapString("service.check")
	conf := &api.Config{
		Address: fmt.Sprintf("%v:%v", consul["host"], consul["port"]),
		Scheme: consul["scheme"],
		Datacenter: consul["datacenter"],
		WaitTime: 3e9,
		Transport: cleanhttp.DefaultPooledTransport(),
	}
	client, err := api.NewClient(conf)
	if err != nil {
		log.Fatal("consul client error :", err)
	}

	if ssl {
		scheme = "https://"
	} else {
		scheme = "http://"
	}

	registration := &api.AgentServiceRegistration{
		ID: viper.GetString("service.id"),
		Name: viper.GetString("service.name"),
		Port: port,
		Tags: []string{"service"},
		Address: addr,
		Check: &api.AgentServiceCheck{
			HTTP: fmt.Sprintf("%v%v:%v%v", scheme, addr, port, check["uri"]),
			Timeout: check["timeout"],
			Interval: check["interval"],
			DeregisterCriticalServiceAfter: check["deregister"],
		},
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		log.Fatal("register server error :", err)
	}
}