package runtimes

func init() {
	// Single-node dev mode; xpack security off keeps the first-run usable.
	register(desc{
		Type:  "elastic",
		Image: "elasticsearch:%s",
		Ports: map[string]string{"9200": "9200"},
		Env: map[string]string{
			"discovery.type":         "single-node",
			"xpack.security.enabled": "false",
			"ES_JAVA_OPTS":           "-Xms512m -Xmx512m",
		},
	}, "es")
}
