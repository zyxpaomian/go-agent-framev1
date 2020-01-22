package future

import (
	"service/etcdclient"
)

func MannualCollect(collectitems []string) {
	localip := Osinfo.Localip
	var err error
	var collectresult string
	for _, collectitem := range collectitems {
		collectresult, err = ScriptExec("collect.py", []string{collectitem})
		if err != nil {
			collectresult = "Error"
		}
		etcdclient.Etcdclient.ClientDiscovery(localip, collectitem, collectresult)
	}
}
