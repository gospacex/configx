package standard

type ElasticsearchConfig struct {
    Addresses []string           `json:"addresses"`
    Username  string             `json:"username"`
    Password  string             `json:"password"`
    Pool      ElasticsearchPool `json:"pool"`
    TLS       ElasticsearchTLS  `json:"tls"`
}

type ElasticsearchPool struct {
    MaxSize int `json:"max_size"`
}

type ElasticsearchTLS struct {
    Enabled bool   `json:"enabled"`
    CAFile  string `json:"ca_file"`
}
