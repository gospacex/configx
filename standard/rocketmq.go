package standard

type RocketMQConfig struct {
    Namesrv   []string `json:"namesrv"`
    Topic     string   `json:"topic"`
    Group     string   `json:"consumer_group"`
    AccessKey string   `json:"access_key"`
    SecretKey string   `json:"secret_key"`
}
