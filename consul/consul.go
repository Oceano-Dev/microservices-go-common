package consul

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/oceano-dev/microservices-go-common/config"

	consul "github.com/hashicorp/consul/api"
)

type ConsulClient struct {
	config *config.Config
	client *consul.Client
}

func NewConsulClient(
	config *config.Config,
	client *consul.Client,
) *ConsulClient {
	consulConfig := consul.DefaultConfig()
	newClient, err := consul.NewClient(consulConfig)
	if err != nil {
		log.Fatalln(err)
	}

	return &ConsulClient{
		config: config,
		client: newClient,
	}
}

func (c *ConsulClient) Register() error {
	port, err := strconv.Atoi(c.config.ListenPort)
	if err != nil {
		return err
	}

	serviceID := c.config.AppName
	address := getHostName()

	registration := &consul.AgentServiceRegistration{
		ID:      serviceID,
		Name:    c.config.AppName,
		Port:    port,
		Address: address,
		Check: &consul.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("https://%s:%v/healthy", address, port),
			Interval:                       "10s",
			Timeout:                        "30s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}

	registrationErr := c.client.Agent().ServiceRegister(registration)

	if registrationErr != nil {
		log.Printf("Failed consul to register service: %s:%v ", address, port)
		return err
	}

	log.Printf("successfully consul register service: %s:%v", address, port)

	return err
}

func (c *ConsulClient) Healthy() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, "Consul check")
	}
}

func getHostName() string {
	hostname, _ := os.Hostname()

	return hostname
}