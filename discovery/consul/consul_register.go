package consul

import (
	consulapi "github.com/hashicorp/consul/api"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"log"
	"time"
	"strconv"
	"github.com/nosixtools/grpc-go-plugins/discovery"
)

type consulRegister struct {
	address string
	ttl    int
}

func NewConsulRegister(address string, ttl int) *consulRegister {
	return &consulRegister{address:address, ttl:ttl}
}

func (cr *consulRegister) Register(info discovery.RegisterInfo) error {

	// initial consul client config
	config := consulapi.DefaultConfig()
	config.Address = cr.address
	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Println("create consul client error:", err.Error())
	}

	serviceId := generateServiceId(info.ServiceName, info.Host, info.Port)

	reg := &consulapi.AgentServiceRegistration{
		ID:      serviceId,
		Name:    info.ServiceName,
		Tags:    []string{info.ServiceName},
		Port:    info.Port,
		Address: info.Host,
	}

	if err = client.Agent().ServiceRegister(reg); err != nil {
		panic(err)
	}

	// initial register service check
	check := consulapi.AgentServiceCheck{TTL: fmt.Sprintf("%ds", cr.ttl), Status: consulapi.HealthPassing}
	err = client.Agent().CheckRegister(
		&consulapi.AgentCheckRegistration{
			ID: serviceId,
			Name: info.ServiceName,
			ServiceID: serviceId,
			AgentServiceCheck: check})
	if err != nil {
		return fmt.Errorf("consul: initial register service check to consul error: %s", err.Error())
	}

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
		x := <-ch
		log.Println("consul: receive signal: ", x)
		// un-register service
		cr.DeRegister(info)

		s, _ := strconv.Atoi(fmt.Sprintf("%d", x))
		os.Exit(s)
	}()

	go func() {
		ticker := time.NewTicker(info.UpdateInterval)
		for {
			<-ticker.C
			err = client.Agent().UpdateTTL(serviceId, "", check.Status)
			if err != nil {
				log.Println("consul: update ttl of service error: ", err.Error())
			}
		}
	}()

	return nil
}

func (cr *consulRegister) DeRegister(info discovery.RegisterInfo) error {

	serviceId := generateServiceId(info.ServiceName, info.Host, info.Port)

	config := consulapi.DefaultConfig()
	config.Address = cr.address
	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Println("create consul client error:", err.Error())
	}

	err = client.Agent().ServiceDeregister(serviceId)
	if err != nil {
		log.Println("consul: deregister service error: ", err.Error())
	} else {
		log.Println("consul: deregistered service from consul server.")
	}

	err = client.Agent().CheckDeregister(serviceId)
	if err != nil {
		log.Println("consul: deregister check error: ", err.Error())
	}

	return nil
}

func generateServiceId(name, host string , port int) string  {
	return fmt.Sprintf("%s-%s-%d", name, host, port)
}
