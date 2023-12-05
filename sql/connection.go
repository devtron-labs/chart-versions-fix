package sql

import (
	"github.com/caarlos0/env"
	"github.com/go-pg/pg"
)

type Config struct {
	Addr            string `env:"PG_ADDR" envDefault:"127.0.0.1"`
	Port            string `env:"PG_PORT" envDefault:"5432"`
	User            string `env:"PG_USER" envDefault:"user"`
	Password        string `env:"PG_PASSWORD" envDefault:"password" secretData:"-"`
	Database        string `env:"PG_DATABASE" envDefault:"orchestrator"`
	ApplicationName string `env:"APP" envDefault:"chart-sync"`
	LogQuery        bool   `env:"PG_LOG_QUERY" envDefault:"false"`
}

func GetConfig() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	return cfg, err
}

func NewDbConnection(cfg *Config) (*pg.DB, error) {
	options := pg.Options{
		Addr:            cfg.Addr + ":" + cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		Database:        cfg.Database,
		ApplicationName: cfg.ApplicationName,
	}
	dbConnection := pg.Connect(&options)
	//check db connection
	var test string
	_, err := dbConnection.QueryOne(&test, `SELECT 1`)

	//--------------
	return dbConnection, err
}

//TODO: call it from somewhere
/*func closeConnection() error {
	return dbConnection.Close()
}*/
