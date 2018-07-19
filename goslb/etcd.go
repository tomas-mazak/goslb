package goslb

import (
	"time"
	"context"
	"github.com/coreos/etcd/clientv3"
	"fmt"
	"encoding/json"
)

var (
    etcdDialTimeout    = 5 * time.Second
    etcdRequestTimeout = 120 * time.Second
)

type EtcdClient struct {
	cli *clientv3.Client
	kv clientv3.KV
}

func (c *EtcdClient) Close() {
	c.cli.Close()
}

func (c *EtcdClient) ListServices() ([][]byte, error) {
	gr, err := etcdClient.kv.Get(c.context(), "/services/", clientv3.WithPrefix())
	if err != nil {
		log.WithError(err).Error("Failed to list services in etcd")
		return nil, err
	}
	ret := make([][]byte, len(gr.Kvs))
	for i, v := range gr.Kvs {
		ret[i] = v.Value
	}
	return ret, nil
}

func (c *EtcdClient) SaveService(service *Service) error {
	doc, err := json.Marshal(service)
	if err != nil {
		log.WithError(err).Error("Failed to serialize service object", service.Domain)
		return err
	}
	ret, err := c.kv.Put(c.context(), fmt.Sprintf("/services/%v", service.Domain), string(doc))
	if err != nil {
		log.WithError(err).Errorf("Failed to store service object in etcd: %v", service.Domain)
		return err
	}
	log.Debugf("Service %v saved to etcd: %v", service.Domain, ret)
	return nil
}

func (c *EtcdClient) DeleteService(name string) error {
	if _, err := c.kv.Delete(c.context(), fmt.Sprintf("/services/%v", name)); err != nil {
		log.WithError(err).Errorf("Failed to delete service object from etcd: %v", name)
		return err
	}
	return nil
}

func (c *EtcdClient) context() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), etcdRequestTimeout)
	return ctx
}

var etcdClient EtcdClient

func InitEtcdClient(config *Config) {
	log.Infof("Connecting to etcd backend %v", config.EtcdServers)
	var err error
	etcdClient.cli, err = clientv3.New(clientv3.Config{
		Endpoints: config.EtcdServers,
		DialTimeout: etcdDialTimeout,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to etcd")
	}
	etcdClient.kv = clientv3.NewKV(etcdClient.cli)
}
