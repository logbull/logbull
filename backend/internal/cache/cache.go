package cache

import (
	"crypto/tls"
	"logbull/internal/config"
	"sync"

	"github.com/valkey-io/valkey-go"
)

var (
	once         sync.Once
	valkeyClient valkey.Client
)

func GetCache() valkey.Client {
	once.Do(func() {
		env := config.GetEnv()

		options := valkey.ClientOption{
			InitAddress: []string{env.ValkeyHost + ":" + env.ValkeyPort},
			Password:    env.ValkeyPassword,
			Username:    env.ValkeyUsername,
		}

		if env.ValkeyIsSsl {
			options.TLSConfig = &tls.Config{
				ServerName: env.ValkeyHost,
			}
		}

		client, err := valkey.NewClient(options)
		if err != nil {
			panic(err)
		}

		valkeyClient = client
	})

	return valkeyClient
}
