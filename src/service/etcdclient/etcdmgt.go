package etcdclient

import (
	"context"
	"time"
	log "util/agentlog"
	ce "util/error"

	"go.etcd.io/etcd/clientv3"
)

type EtcdClient struct {
	Ip      string
	Leaseid clientv3.LeaseID
	Client  *clientv3.Client
	Lease   clientv3.Lease
}

var Etcdclient EtcdClient

func (e *EtcdClient) ClientInit(ip string, endpoints []string) {
	cli, _ := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
	})

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := cli.Status(timeoutCtx, endpoints[0])
	if err != nil {
		panic("初始化Etcd失败")
	}

	// 创建租约
	lease := clientv3.NewLease(cli)

	//设置租约时间
	leaseResp, err := lease.Grant(context.TODO(), 30)
	if err != nil {
		log.Errorf("设置租约时间失败, 失败原因: %s", err.Error())
		panic("设置租约时间失败")
	}

	// 设置续租
	leaseID := leaseResp.ID

	e.Ip = ip
	e.Client = cli
	e.Leaseid = leaseID
	e.Lease = lease
}

func (e *EtcdClient) GetSingleCfg(url string) (string, error) {
	getResp, err := e.Client.Get(context.TODO(), url, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}
	if len(getResp.Kvs) == 0 {
		return "", ce.New("etcd取值不存在该Key")
	}

	return string((getResp.Kvs[0]).Value), nil
	// if _, ok := getResp.Kvs[0]["Value"]; ok {
	//return string((getResp.Kvs[0]).Value), nil
	//}
	//return "", ce.New("etcd取值不存在该Key")
}

func (e *EtcdClient) WatchCfg(url string, wchchan chan string) {
	wch := e.Client.Watch(context.Background(), url)
	for item := range wch {
		wchchan <- string((item.Events[0]).Kv.Value)
	}
	close(wchchan)
}

func (e *EtcdClient) ClientDiscovery(ip string, infoname string, infodata string) {
	var keepResp *clientv3.LeaseKeepAliveResponse
	var keepRespChan <-chan *clientv3.LeaseKeepAliveResponse

	key := "/server/" + ip + "/" + infoname
	_, err := e.Client.Put(context.TODO(), key, infodata, clientv3.WithLease(e.Leaseid))
	if err != nil {
		log.Errorf("存入首次获取的信息失败: %s", err.Error)
		panic(err)
	}
	if keepRespChan, err = e.Lease.KeepAlive(context.TODO(), e.Leaseid); err != nil {
		log.Errorf("创建自动续期失败，失败原因: %s", err.Error)
		panic(err)
	}
	go func() {
		for {
			select {
			case keepResp = <-keepRespChan:
				if keepRespChan == nil {
					log.Errorln("自动续期失败")
				} else { //每秒会续租一次，所以就会受到一次应答
					log.Debugf("收到自动续租应答: %v", keepResp.ID)
				}
			}
		}
	}()
}

func (e *EtcdClient) ClientPut(keyurl string, value string) (string, error) {
	_, err := e.Client.Put(context.TODO(), keyurl, value)
	if err != nil {
		log.Errorf("存入任务执行信息失败: %s", err.Error())
		return "", err
	}
	return "Success", nil

}
