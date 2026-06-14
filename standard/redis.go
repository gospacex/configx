package standard

import "time"

type RedisConfig struct {
    Addr     string        `json:"addr"`
    Password string        `json:"password"`
    DB       int           `json:"db"`
    Cluster  RedisCluster `json:"cluster"`
    Sentinel RedisSentinel `json:"sentinel"`
    Pool     RedisPool     `json:"pool"`
}

type RedisCluster struct {
    Servers []string `json:"servers"`
}

type RedisSentinel struct {
    Master string   `json:"master"`
    Nodes  []string `json:"nodes"`
}

type RedisPool struct {
    MaxActive       int           `json:"max_active"`
    MaxIdle         int           `json:"max_idle"`
    MaxIdleDuration time.Duration `json:"max_idle_duration"`
}
