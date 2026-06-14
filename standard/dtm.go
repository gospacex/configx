package standard

import "time"

type DTMConfig struct {
    Endpoint string        `json:"endpoint"`
    Timeout  time.Duration `json:"timeout"`
}
