package main

type Controller struct {
	Name      string     `json:"name"`
	Enqueue   string     `json:"enqueue"`
	Retry     int        `json:"retryOnError"`
	Resources []Resource `json:"resources"`
}

type Resource struct {
	Group     string
	Version   string
	Kind      string
	Namespace string
}

type Import struct {
	Alias string
	Pkg   string
}
