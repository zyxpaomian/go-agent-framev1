package structs

type AgentInfo struct {
	Agentip  string `json:agentip`
	Alive    int    `json:alive`
	Dpswitch int    `json:dpswitch`
}

type PingMesh struct {
	Dstip    string `json:dstip`
	Durtime  int64  `json:durtime`
	Status   string `json:status`
	Exectime string `json:exectime`
}

type TcpMesh struct {
	Dstip    string `json:dstip`
	Status   string `json:status`
	Exectime string `json:exectime`
	Port     string    `json:port`
}
