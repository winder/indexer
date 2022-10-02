package blockprocessor

// Config for a block processor
type Config struct {
	// Catchpoint to initialize the local ledger to
	Catchpoint string `yaml:"catchpoint"`

	IndexerDatadir string `yaml:"indexer-data-dir"`
	AlgodDataDir   string `yaml:"algod-data-dir"`
	AlgodToken     string `yaml:"algod-token"`
	AlgodAddr      string `yaml:"algod-addr"`

	// OverrideNextRound
	OverrideNextRound uint64 `yaml:"override-next-round"`
}
