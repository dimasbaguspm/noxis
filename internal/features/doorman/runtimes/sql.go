package runtimes

func init() {
	register(desc{
		Type:  "postgres",
		Image: "postgres:%s",
		Ports: map[string]string{"5432": "5432"},
		Env:   map[string]string{"POSTGRES_PASSWORD": "noxis"},
	}, "pg")

	register(desc{
		Type:  "mysql",
		Image: "mysql:%s",
		Ports: map[string]string{"3306": "3306"},
		Env:   map[string]string{"MYSQL_ROOT_PASSWORD": "noxis"},
	})

	register(desc{
		Type:  "mariadb",
		Image: "mariadb:%s",
		Ports: map[string]string{"3306": "3306"},
		Env:   map[string]string{"MARIADB_ROOT_PASSWORD": "noxis"},
	})
}
