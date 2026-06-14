package standard

type KafkaConfig struct {
    Brokers []string  `json:"brokers"`
    Topic   string   `json:"topic"`
    Group   string   `json:"consumer_group"`
    SASL    KafkaSASL `json:"sasl"`
}

type KafkaSASL struct {
    Mechanism string `json:"mechanism"`
    User      string `json:"user"`
    Password  string `json:"password"`
}
