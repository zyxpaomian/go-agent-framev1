package agentctrl

import (
	"dao/agentdao"
	"structs"
	ce "util/error"
	log "util/serverlog"
)

func GetAgentInfo() ([]*structs.AgentInfo, error) {
	tasklist := []*structs.AgentInfo{}
	tasklist, err := agentdao.AgentDao.GetAgentInfo()
	if err != nil {
		log.Errorln("拉取所有客户端清单信息失败")
		return nil, ce.GetRsyncTaskError()
	}
	return tasklist, nil
}

func UpdateAgentAlive(alivestate int, agentip string) error {
	_, err := agentdao.AgentDao.UpdateAgentAlive(alivestate, agentip)
	if err != nil {
		log.Errorln("更新客户端主机状态失败")
		return ce.GetRsyncTaskError()
	}
	return nil
}
