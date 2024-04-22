package models

type LogInfo struct {
	Timestamp string `json:"timestamp"`
	JsonLog   string `json:"json"`
}

func (l LogInfo) String() string {
	return l.Timestamp + " " + l.JsonLog + "\n"
}
