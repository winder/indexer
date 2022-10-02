package postgresql

// serde for converting an Config to/from a PostgresqlExporterConfig

// Config specific to the postgresql exporter
type Config struct {
	// Pgsql connection string
	// See https://github.com/jackc/pgconn for more details
	ConnectionString string `yaml:"connection-string"`
	// Maximum connection number for connection pool
	// This means the total number of active queries that can be running
	// concurrently can never be more than this
	MaxConn uint32 `yaml:"max-conn"`
	// TODO: this should be less visible.
	// The test flag will replace an actual DB connection being created via the connection string,
	// with a mock DB for unit testing.
	Test bool `yaml:"test"`

	// TODO: Pass round override to init function.
	RoundOverride uint64 `yaml:"round-override"`
}
