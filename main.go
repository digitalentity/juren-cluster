package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"juren/commands"
	"juren/config"
	"juren/datastore/block"
	"juren/oid"
	"os"

	"github.com/fxamacker/cbor/v2"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func setLogLevel(level string) {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(l)
}

func RunPublish(ctx context.Context, cfg *config.Config) {
	log.Infof("Publishing ipfs-go-storage...")
}

func RunTest(ctx context.Context, cfg *config.Config) {
	log.Infof("Running test for ipfs-go-storage...")

	testString := "/Anime/Boku no Pico/[RAW]/Boku no Pico - 01 (v2) [720p].mkv"
	sha256Hash := sha256.Sum256([]byte(testString))

	o, err := oid.Encode(oid.OidTypeRawData, sha256Hash)
	if err != nil {
		log.Fatalf("Failed to encode OID: %v", err)
	}

	log.Infof("Encoded OID: %s", o.String())

	o2, err := oid.FromString(o.String())
	if err != nil {
		log.Fatalf("Failed to decode OID: %v", err)
	}

	log.Infof("Decoded OID: %s, type: %d", o2.String(), o2.Type())

	b, err := cbor.Marshal(o)
	if err != nil {
		log.Fatalf("Failed to encode OID to CBOR: %v", err)
	}
	log.Infof("CBOR encoded OID: %s", hex.EncodeToString(b))

	o3 := &oid.Oid{}
	if err := cbor.Unmarshal(b, o3); err != nil {
		log.Fatalf("Failed to decode CBOR OID: %v", err)
	}
	log.Infof("Decoded CBOR OID: %s, type: %d", o3.String(), o3.Type())

	blk := &block.Block{
		Oid:    *o,
		Length: uint64(len(testString)),
	}

	b, err = cbor.Marshal(blk)
	if err != nil {
		log.Fatalf("Failed to encode Block to CBOR: %v", err)
	}
	log.Infof("CBOR encoded Block: %s", hex.EncodeToString(b))

}

func registerGlobalFlags(fset *flag.FlagSet) {
	flag.VisitAll(func(f *flag.Flag) {
		fset.Var(f.Value, f.Name, f.Usage)
	})
}

// main is the entry point of the application.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configFile := flag.String("config", "/tmp/juren-cluster/config.json", "Path to config file")
	logLevel := flag.String("loglevel", "info", "Log level")

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	registerGlobalFlags(initCmd)

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	mountPoint := serveCmd.String("mountpoint", "/tmp/juren-cluster/mount", "Path to mount the filesystem at")
	registerGlobalFlags(serveCmd)

	publishCmd := flag.NewFlagSet("publish", flag.ExitOnError)
	registerGlobalFlags(publishCmd)

	testCmd := flag.NewFlagSet("test", flag.ExitOnError)
	registerGlobalFlags(testCmd)

	if len(os.Args) < 2 {
		log.WithField("args", os.Args).Fatal("Expected a subcommand")
	}
	cmd, args := os.Args[1], os.Args[2:]

	switch cmd {
	case "init":
		initCmd.Parse(args)
		setLogLevel(*logLevel)
		cfg := config.NewEmptyConfig(*configFile)
		commands.RunInit(ctx, cfg)
	case "serve":
		serveCmd.Parse(args)
		setLogLevel(*logLevel)
		cfg, err := config.NewConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		commands.RunServe(ctx, cfg, *mountPoint)
	case "publish":
		publishCmd.Parse(args)
		setLogLevel(*logLevel)
		cfg, err := config.NewConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		RunPublish(ctx, cfg)
	case "test":
		testCmd.Parse(args)
		setLogLevel(*logLevel)
		RunTest(ctx, nil)
	default:
		log.Fatalf("Invalid subcommand '%s'", os.Args[1])
	}
}
