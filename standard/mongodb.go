package standard

type MongoDBConfig struct {
    URI        string           `json:"uri"`
    ReplicaSet MongoDBReplicaSet `json:"replica_set"`
    Auth       MongoDBAuth      `json:"auth"`
    Pool       MongoDBPool      `json:"pool"`
}

type MongoDBReplicaSet struct {
    Members   []string `json:"members"`
    Database string   `json:"database"`
}

type MongoDBAuth struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type MongoDBPool struct {
    MaxSize int `json:"max_size"`
    MinSize int `json:"min_size"`
}
