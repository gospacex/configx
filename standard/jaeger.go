package standard

type JaegerConfig struct {
    Endpoint    string        `json:"endpoint"`
    ServiceName string      `json:"service_name"`
    Sampler     JaegerSampler `json:"sampler"`
}

type JaegerSampler struct {
    Type string  `json:"type"`
    Rate float64 `json:"rate"`
}
