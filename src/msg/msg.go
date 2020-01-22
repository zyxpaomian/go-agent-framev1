package msg

import (
	"github.com/golang/protobuf/proto"
)

const HeartBeat_Msg = 1
const HeartBeat_Rsp = 2
const ShellRun_Msg = 3
const ShellRun_Rsp = 4
const FileTrans_Msg = 5
const ScriptRun_Msg = 6
const ScriptRun_Rsp = 7

type Msg struct {
	// MsgLength uint64
	MsgType  uint32
	MsgData  []byte
	MsgProto proto.Message
}

//type ProtoMsg struct {
//	Msg proto.Message
//}
