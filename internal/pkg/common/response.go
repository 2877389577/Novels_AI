package common

type Response struct {
	// 状态码
	Code int `json:"code"`
	// 状态描述
	Msg string `json:"msg"`
	// 数据
	Data interface{} `json:"data"`
}
