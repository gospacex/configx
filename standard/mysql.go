package standard

import "time"

type MySQLConfig struct {
	DSN            string       `json:"dsn"`
	Cluster        MySQLCluster `json:"cluster"`
	Pool           MySQLPool    `json:"pool"`
	ReadWriteSplit bool         `json:"read_write_split"`
}

type MySQLCluster struct {
	Servers []string `json:"servers"`
	Mode    string   `json:"mode"`
}

type MySQLPool struct {
	MaxOpenConns int           `json:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns"`
	ConnMaxLife  time.Duration `json:"conn_max_life"`
	MaxIdleTime  time.Duration `json:"max_idle_time"`
}

/**
场景	推荐配置	原因
开发/测试环境	DSN 方式	简单直接
生产单机	cluster + mode:"single"	可统一管理连接池
生产主从	cluster + mode:"master-slave"	支持读写分离
生产集群	cluster + mode:"galera"	高可用 + 强一致性
*/
