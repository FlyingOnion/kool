

```go
import "k8s.io/client-go/rest"
import "k8s.io/client-go/tools/clientcmd"

// 1. get an REST client

// 1.1 get an REST config

// use default InclusterConfig
config := rest.InclusterConfig()

// or parse a config using kubeconfig file path and master url
var kubeconfig string
var master string
config := clientcmd.BuildConfigFromFlags(master, kubeconfig)

// 1.2 get an http client, and construct a REST client

client, err := rest.RESTClientFor(config) (*RESTClient, error)

// or
httpClient, err := rest.HTTPClientFor(config)
client, err := rest.RESTClientForConfigAndClient(config, httpClient)

// 2. 
```