package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/algorand/go-algorand/data/bookkeeping"
	"github.com/algorand/go-algorand/ledger/ledgercore"

	"github.com/algorand/indexer/data"
	"github.com/algorand/indexer/exporters"
	"github.com/algorand/indexer/idb"
	// Necessary to ensure the postgres implementation has been registered in the idb factory
	_ "github.com/algorand/indexer/idb/postgres"
	"github.com/algorand/indexer/importer"
	"github.com/algorand/indexer/plugins"
)

const exporterName = "postgresql"

type postgresqlExporter struct {
	round  uint64
	cfg    Config
	db     idb.IndexerDb
	logger *logrus.Logger
}

var postgresqlExporterMetadata = exporters.ExporterMetadata{
	ExpName:        exporterName,
	ExpDescription: "Exporter for writing data to a postgresql instance.",
	ExpDeprecated:  false,
}

// Constructor is the ExporterConstructor implementation for the "postgresql" exporter
type Constructor struct{}

// New initializes a postgresqlExporter
func (c *Constructor) New() exporters.Exporter {
	return &postgresqlExporter{
		round: 0,
	}
}

func (exp *postgresqlExporter) Metadata() exporters.ExporterMetadata {
	return postgresqlExporterMetadata
}

func (exp *postgresqlExporter) Init(_ context.Context, cfg plugins.PluginConfig, logger *logrus.Logger) error {
	dbName := "postgres"
	exp.logger = logger
	var err error
	exp.cfg, err = exp.unmarshalConfig(string(cfg))
	if err != nil {
		return fmt.Errorf("Init() connect failure in unmarshalConfig: %v", err)
	}
	// Inject a dummy db for unit testing
	if exp.cfg.Test {
		dbName = "dummy"

	}
	var opts idb.IndexerDbOptions
	opts.MaxConn = exp.cfg.MaxConn
	opts.ReadOnly = false
	db, ready, err := idb.IndexerDbByName(dbName, exp.cfg.ConnectionString, opts, exp.logger)
	if err != nil {
		return fmt.Errorf("Init() connect failure constructing db, %s: %v", dbName, err)
	}
	exp.db = db
	<-ready

	rnd, err := exp.db.GetNextRoundToAccount()
	if err == nil || err == idb.ErrorNotInitialized {
		exp.round = rnd
		err = nil
	} else {
		return fmt.Errorf("Init() failed to get next roun: %w", err)
	}

	if exp.cfg.RoundOverride != 0 {
		exp.round = exp.cfg.RoundOverride
		err = exp.db.SetNextRoundToAccount(exp.cfg.RoundOverride)
	}
	return err
}

func (exp *postgresqlExporter) Config() plugins.PluginConfig {
	ret, _ := yaml.Marshal(exp.cfg)
	return plugins.PluginConfig(ret)
}

func (exp *postgresqlExporter) Close() error {
	if exp.db != nil {
		exp.db.Close()
	}
	return nil
}

func (exp *postgresqlExporter) Receive(exportData data.BlockData) error {
	start := time.Now()
	if exportData.Delta == nil {
		if exportData.Round() == 0 {
			exportData.Delta = &ledgercore.StateDelta{}
		} else {
			return fmt.Errorf("receive got an invalid block: %#v", exportData)
		}
	}
	// Do we need to test for consensus protocol here?
	/*
		_, ok := config.Consensus[block.CurrentProtocol]
			if !ok {
				return fmt.Errorf("protocol %s not found", block.CurrentProtocol)
		}
	*/
	var delta ledgercore.StateDelta
	if exportData.Delta != nil {
		delta = *exportData.Delta
	}
	vb := ledgercore.MakeValidatedBlock(
		bookkeeping.Block{
			BlockHeader: exportData.BlockHeader,
			Payset:      exportData.Payset,
		},
		delta)
	if err := exp.db.AddBlock(&vb); err != nil {
		return err
	}
	exp.round = exportData.Round() + 1
	exp.logger.Infof("Receive() round exported (%s)", time.Since(start))

	return nil
}

func (exp *postgresqlExporter) HandleGenesis(genesis bookkeeping.Genesis) error {
	_, err := importer.EnsureInitialImport(exp.db, genesis)
	return err
}

func (exp *postgresqlExporter) Round() uint64 {
	// should we try to retrieve this from the db? That could fail.
	// return exp.db.GetNextRoundToAccount()
	return exp.round
}

func (exp *postgresqlExporter) unmarshalConfig(cfg string) (Config, error) {
	var config Config
	err := yaml.Unmarshal([]byte(cfg), &config)
	return config, err
}

func init() {
	exporters.RegisterExporter(exporterName, &Constructor{})
}
