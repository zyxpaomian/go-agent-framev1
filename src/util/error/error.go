package error

type MyError struct {
	msg string
}

func (e *MyError) Error() string {
	return e.msg
}

func New(msg string) *MyError {
	return &MyError{msg: msg}
}

func DBError() *MyError {
	return New("数据库错误")
}

func AuthError() *MyError {
	return New("用户认证错误")
}

func GetCronError() *MyError {
	return New("获取计划任务失败")
}

func GetSyncTaskError() *MyError {
	return New("获取同步任务列表失败")
}

func GetRsyncTaskError() *MyError {
	return New("获取异步任务列表失败")
}
func GetRsyncTaskResult() *MyError {
	return New("获取异步任务结果失败")
}


