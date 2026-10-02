package runtimes

func init() {
	register(desc{
		Type:  "redis",
		Image: "redis:%s",
		Ports: map[string]string{"6379": "6379"},
	})
}
