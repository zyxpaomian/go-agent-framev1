package influx

import (
	"service/etcdclient"
	"time"
	log "util/agentlog"

	"github.com/influxdata/influxdb/client/v2"
)

func InfluxInit() (client.Client, error) {
	bind, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/influxdb/bind")
	username, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/influxdb/username")
	password, _ := etcdclient.Etcdclient.GetSingleCfg("/server/cfg/influxdb/password")
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     bind,
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Errorln("influxdb初始化失败")
		return nil, err
	}
	return cli, nil
}

func SimpleInsert(cli client.Client, db string, tb string, tags map[string]string, fields map[string]interface{}) {
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  db,
		Precision: "s",
	})
	if err != nil {
		log.Errorf("连接Influxdb失败，失败原因: %s", err.Error())
	}

	pt, err := client.NewPoint(
		tb,
		tags,
		fields,
		time.Now(),
	)
	if err != nil {
		log.Errorf("连接Influxdb失败，失败原因: %s", err.Error())
	}
	bp.AddPoint(pt)

	err = cli.Write(bp)
	if err != nil {
		log.Errorf("写入Influxdb失败，失败原因: %s", err.Error())
	}

	if err = cli.Close(); err != nil {
		log.Errorf("关闭Influxdb失败，失败原因: %s", err.Error())
	}

}
