package runtimes

func init() {
	// Runs without auth by default; add MONGO_INITDB_ROOT_USERNAME/PASSWORD
	// via spec env when needed.
	register(desc{
		Type:  "mongo",
		Image: "mongo:%s",
		Ports: map[string]string{"27017": "27017"},
	})
}
