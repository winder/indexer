package filewriter_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/data/bookkeeping"
	"github.com/algorand/go-algorand/ledger/ledgercore"

	"github.com/algorand/indexer/data"
	"github.com/algorand/indexer/exporters"
	"github.com/algorand/indexer/exporters/filewriter"
	"github.com/algorand/indexer/plugins"
)

var logger *logrus.Logger
var fileCons = &filewriter.Constructor{}
var configNoDelta = "block-dir: %s/blocks\nexclude-state-delta: true\n"
var configWithDelta = "block-dir: %s/blocks\nexclude-state-delta: false\n"

func init() {
	logger, _ = test.NewNullLogger()
}

func insertTempDir(config string, dir string) string {
	return fmt.Sprintf(config, dir)
}

func TestExporterMetadata(t *testing.T) {
	fileExp := fileCons.New()
	meta := fileExp.Metadata()
	assert.Equal(t, plugins.PluginType(plugins.Exporter), meta.Type())
	assert.Equal(t, "filewriter", meta.Name())
	assert.Equal(t, "Exporter for writing data to a file.", meta.Description())
	assert.Equal(t, false, meta.Deprecated())
}

func TestExporterInit(t *testing.T) {
	tempdir := t.TempDir()
	config := insertTempDir(configNoDelta, tempdir)
	fileExp := fileCons.New()
	assert.Equal(t, uint64(0), fileExp.Round())
	// creates a new output file
	err := fileExp.Init(plugins.PluginConfig(config), logger)
	assert.NoError(t, err)
	pluginConfig := fileExp.Config()
	assert.Equal(t, config, string(pluginConfig))
	assert.Equal(t, uint64(0), fileExp.Round())
	fileExp.Close()
	// can open existing file
	err = fileExp.Init(plugins.PluginConfig(config), logger)
	assert.NoError(t, err)
	fileExp.Close()
	// re-initializes empty file
	path := fmt.Sprintf("%s/blocks/metadata.json", tempdir)
	assert.NoError(t, os.Remove(path))
	f, err := os.Create(path)
	f.Close()
	assert.NoError(t, err)
	err = fileExp.Init(plugins.PluginConfig(config), logger)
	assert.NoError(t, err)
	fileExp.Close()
}

func TestExporterHandleGenesis(t *testing.T) {
	tempdir := t.TempDir()
	config := insertTempDir(configNoDelta, tempdir)

	fileExp := fileCons.New()
	fileExp.Init(plugins.PluginConfig(config), logger)
	genesisA := bookkeeping.Genesis{
		SchemaID:    "test",
		Network:     "test",
		Proto:       "test",
		Allocation:  nil,
		RewardsPool: "AAAAAAA",
		FeeSink:     "AAAAAAA",
		Timestamp:   1234,
		Comment:     "",
		DevMode:     true,
	}
	err := fileExp.HandleGenesis(genesisA)
	fileExp.Close()
	assert.NoError(t, err)
	metadataFile := fmt.Sprintf("%s/blocks/metadata.json", tempdir)
	require.FileExists(t, metadataFile)
	configs, err := ioutil.ReadFile(metadataFile)
	assert.NoError(t, err)
	var blockMetaData filewriter.BlockMetaData
	err = json.Unmarshal(configs, &blockMetaData)
	assert.Equal(t, uint64(0), blockMetaData.NextRound)
	assert.Equal(t, string(genesisA.Network), blockMetaData.Network)
	assert.Equal(t, crypto.HashObj(genesisA).String(), blockMetaData.GenesisHash)

	// genesis mismatch
	fileExp.Init(plugins.PluginConfig(config), logger)
	genesisB := bookkeeping.Genesis{
		SchemaID:    "test",
		Network:     "test",
		Proto:       "test",
		Allocation:  nil,
		RewardsPool: "AAAAAAA",
		FeeSink:     "AAAAAAA",
		Timestamp:   5678,
		Comment:     "",
		DevMode:     false,
	}

	err = fileExp.HandleGenesis(genesisB)
	assert.Contains(t, err.Error(), "genesis hash in metadata does not match expected value")
	fileExp.Close()
}

func sendData(t *testing.T, fileExp exporters.Exporter, config string, numRounds int) {
	block := data.BlockData{
		BlockHeader: bookkeeping.BlockHeader{
			Round: 3,
		},
		Payset:      nil,
		Delta:       nil,
		Certificate: nil,
	}
	// exporter not initialized
	err := fileExp.Receive(block)
	assert.Contains(t, err.Error(), "exporter not initialized")

	// initialize
	err = fileExp.Init(plugins.PluginConfig(config), logger)
	require.NoError(t, err)

	// incorrect round
	err = fileExp.Receive(block)
	assert.Contains(t, err.Error(), "received round 3, expected round 0")

	// genesis
	genesis := bookkeeping.Genesis{
		SchemaID:    "test",
		Network:     "test",
		Proto:       "test",
		Allocation:  nil,
		RewardsPool: "AAAAAAA",
		FeeSink:     "AAAAAAA",
		Timestamp:   1234,
		Comment:     "",
		DevMode:     true,
	}
	err = fileExp.HandleGenesis(genesis)
	assert.NoError(t, err)

	// write block to file
	for i := 0; i < numRounds; i++ {
		block = data.BlockData{
			BlockHeader: bookkeeping.BlockHeader{
				Round: basics.Round(i),
			},
			Payset:      nil,
			Delta:       &ledgercore.StateDelta{},
			Certificate: nil,
		}
		err = fileExp.Receive(block)
		assert.NoError(t, err)
		assert.Equal(t, uint64(i+1), fileExp.Round())

	}

	assert.NoError(t, fileExp.Close())
}

func TestExporterReceiveNoDelta(t *testing.T) {
	tempdir := t.TempDir()
	config := insertTempDir(configNoDelta, tempdir)
	fileExp := fileCons.New()
	numRounds := 5
	sendData(t, fileExp, config, numRounds)

	// block data is valid
	for i := 0; i < 5; i++ {
		filename := fmt.Sprintf(filewriter.FileExporterFileFormat, "block", i)
		path := fmt.Sprintf("%s/blocks/%s", tempdir, filename)
		assert.FileExists(t, path)
		b, _ := os.ReadFile(path)
		var blockData data.BlockData
		err := json.Unmarshal(b, &blockData)
		assert.NoError(t, err)
	}

	// delta data is not written
	for i := 0; i < 5; i++ {
		filename := fmt.Sprintf(filewriter.FileExporterFileFormat, "delta", i)
		path := fmt.Sprintf("%s/blocks/%s", tempdir, filename)
		assert.NoFileExists(t, path)
	}

	//	should continue from round 6 after restart
	fileExp.Init(plugins.PluginConfig(config), logger)
	assert.Equal(t, uint64(5), fileExp.Round())
	fileExp.Close()
}

func TestExporterReceiveWithDelta(t *testing.T) {
	tempdir := t.TempDir()
	config := insertTempDir(configWithDelta, tempdir)
	fileExp := fileCons.New()
	numRounds := 5
	sendData(t, fileExp, config, numRounds)

	// block data is valid
	for i := 0; i < 5; i++ {
		filename := fmt.Sprintf(filewriter.FileExporterFileFormat, "block", i)
		path := fmt.Sprintf("%s/blocks/%s", tempdir, filename)
		assert.FileExists(t, path)
		b, _ := os.ReadFile(path)
		var blockData data.BlockData
		err := json.Unmarshal(b, &blockData)
		assert.NoError(t, err)
	}

	// delta data is not written
	for i := 0; i < 5; i++ {
		filename := fmt.Sprintf(filewriter.FileExporterFileFormat, "delta", i)
		path := fmt.Sprintf("%s/blocks/%s", tempdir, filename)
		assert.FileExists(t, path)
		b, _ := os.ReadFile(path)
		var blockData data.BlockData
		err := json.Unmarshal(b, &blockData)
		assert.NoError(t, err)
	}

	//	should continue from round 6 after restart
	fileExp.Init(plugins.PluginConfig(configNoDelta), logger)
	assert.Equal(t, uint64(5), fileExp.Round())
	fileExp.Close()
}

func TestMissingStateDelta(t *testing.T) {
	tempdir := t.TempDir()
	config := insertTempDir(configWithDelta, tempdir)
	fileExp := fileCons.New()
	fileExp.Init(plugins.PluginConfig(config), logger)

	block := data.BlockData{
		BlockHeader: bookkeeping.BlockHeader{
			Round: basics.Round(0),
		},
		Payset:      nil,
		Delta:       nil,
		Certificate: nil,
	}
	err := fileExp.Receive(block)
	require.ErrorContains(t, err, "exporter is misconfigured, set 'exclude-state-delta: true' or enable a plugin that provides state deltas data")
}

func TestExporterClose(t *testing.T) {
	tempdir := t.TempDir()
	config := insertTempDir(configNoDelta, tempdir)
	fileExp := fileCons.New()
	fileExp.Init(plugins.PluginConfig(config), logger)
	block := data.BlockData{
		BlockHeader: bookkeeping.BlockHeader{
			Round: 0,
		},
		Payset:      nil,
		Delta:       nil,
		Certificate: nil,
	}
	err := fileExp.Receive(block)
	require.NoError(t, err)
	err = fileExp.Close()
	assert.NoError(t, err)
	metadataFile := fmt.Sprintf("%s/blocks/metadata.json", tempdir)
	require.FileExists(t, metadataFile)
	// assert round is updated correctly
	configs, err := ioutil.ReadFile(metadataFile)
	assert.NoError(t, err)
	var blockMetaData filewriter.BlockMetaData
	err = json.Unmarshal(configs, &blockMetaData)
	assert.Equal(t, uint64(1), blockMetaData.NextRound)
}
