package agentdao

import (
	"structs"
	ce "util/error"
	"util/mysql"
	log "util/serverlog"
)

type AgentMgtDao struct {
}

var AgentDao AgentMgtDao

func (agentdao *AgentMgtDao) GetAgentInfo() ([]*structs.AgentInfo, error) {
	resultlist := []*structs.AgentInfo{}
	tx := mysql.DB.GetTx()
	if tx == nil {
		log.Errorln("MySQL 获取TX失败")
		return nil, ce.DBError()
	}
	stmt, err := tx.Prepare("select agentip,alive,dpswitch from agent_status;")
	if err != nil {
		tx.Rollback()
		log.Errorln("MySQL 获取TX失败: ", err.Error())
		return nil, ce.DBError()
	}
	rows, err := stmt.Query()
	if err != nil {
		log.Errorln("MySQL 查询失败: ", err.Error())
		stmt.Close()
		return nil, ce.DBError()
	}
	for rows.Next() {
		result := &structs.AgentInfo{}
		err := rows.Scan(&result.Agentip, &result.Alive, &result.Dpswitch)
		if err != nil {
			log.Errorln("MySQL 查询失败: ", err.Error())
			rows.Close()
			stmt.Close()
			tx.Rollback()
			return nil, ce.DBError()
		} else {
			resultlist = append(resultlist, result)
		}
	}
	rows.Close()
	stmt.Close()
	tx.Commit()
	return resultlist, nil

}

func (agentdao *AgentMgtDao) UpdateAgentAlive(alivestate int, agentip string) (int, error) {
	updateid, err := mysql.DB.SimpleUpdate("update agent_status set alive = ? where agentip = ?;", alivestate, agentip)
	if err != nil || updateid == -1 {
		log.Errorln("更新客户端状态失败")
		return -1, ce.DBError()
	}
	return updateid, nil
}
