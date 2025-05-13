module github.com/jgulick48/lxp-bridge-go

go 1.24.1

require (
	github.com/DataDog/datadog-go v4.8.3+incompatible
	github.com/eclipse/paho.mqtt.golang v1.5.0
	github.com/mitchellh/panicwrap v1.0.0
	github.com/sigurn/crc16 v0.0.0-20240131213347-83fcde1e29d1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.10.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
)

tool golang.org/x/tools/cmd/stringer
