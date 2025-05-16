package main

import (
	"context"
	"flag"
	"juren/commands"
	"juren/config"
	"os"

	log "github.com/sirupsen/logrus"
)

func setLogLevel(level string) {
	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatalf("Invalid log level: %v", err)
	}
	log.SetLevel(l)
}

// type blockIndex struct {
// }

// func (b *blockIndex) GetByOid(oid.Oid) (*block.ExtendedMedatadata, error) {
// 	return nil, errors.ErrUnsupported
// }

// func (b *blockIndex) GetBySeq(uint64) (*block.ExtendedMedatadata, error) {
// 	return nil, errors.ErrUnsupported
// }

// func (b *blockIndex) Put(oid.Oid, *block.ExtendedMedatadata) (*block.ExtendedMedatadata, error) {
// 	return nil, errors.ErrUnsupported
// }

// func (b *blockIndex) EnumerateBySeq(uint64, uint64) ([]*block.ExtendedMedatadata, error) {
// 	bl := []*block.ExtendedMedatadata{
// 		{
// 			Sequence: 1,
// 			Metadata: &block.Metadata{
// 				Oid:        *oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXT"),
// 				Length:     10,
// 				UpdateTime: time.Now(),
// 				IsDeleted:  false,
// 			},
// 		},
// 		{Sequence: 2,
// 			Metadata: &block.Metadata{
// 				Oid:        *oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXU"),
// 				Length:     20,
// 				UpdateTime: time.Now(),
// 				IsDeleted:  false,
// 			},
// 		},
// 		{
// 			Sequence: 3,
// 			Metadata: &block.Metadata{
// 				Oid:        *oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXV"),
// 				Length:     30,
// 				UpdateTime: time.Now(),
// 				IsDeleted:  true,
// 			},
// 		},
// 	}
// 	return bl, nil
// }

// type DemoServer struct {
// }

// func (d *DemoServer) PeerSync(req *protocol.PeerSyncRequest, res *protocol.PeerSyncResponse) error {
// 	log.Infof("Received PeerSync from %s, seq: %d", req.NodeID.String(), req.SequenceNumber)
// 	return errors.ErrUnsupported
// }

// func (d *DemoServer) PeerAnnouncement(msg *protocol.PeerAnnouncementMessage) {
// 	log.Infof("Received PeerAnnouncement from %s, addr: %s, seq: %d", msg.NodeID.String(), msg.Address, msg.SequenceNumber)
// }

// 	idx, err := ldb.NewBlockIndex("/tmp/juren-cluster/leveldb")
// 	if err != nil {
// 		log.Fatalf("Failed to create LevelDB: %v", err)
// 	}

// 	log.Infof("LevelDB SEQ: %d", idx.GetSeq())

// 	// o := oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXT")
// 	// blk := &block.MetadataWithSeq{
// 	// 	Metadata: &block.Metadata{
// 	// 		Oid:        *o,
// 	// 		Length:     10,
// 	// 		UpdateTime: time.Now(),
// 	// 		IsDeleted:  false,
// 	// 	},
// 	// }

// 	// md, err := idx.Put(blk)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to put metadata: %v", err)
// 	// }
// 	md, err := idx.EnumerateBySeq(0, 100)
// 	if err != nil {
// 		log.Fatalf("Failed to get metadata: %v", err)
// 	}
// 	for _, e := range md {
// 		log.Infof("Put metadata: %s, seq: %d", e.Metadata.Oid.String(), e.Sequence)
// 	}

// 	return

// 	groupAddr, err := net.ResolveUDPAddr("udp", "224.0.0.1:9999")
// 	if err != nil {
// 		log.Fatalf("Failed to resolve UDP address: %v", err)
// 	}

// 	rc, err := net.ListenMulticastUDP("udp", nil, groupAddr)
// 	if err != nil {
// 		log.Fatalf("Failed to listen on UDP: %v", err)
// 	}

// 	wc, err := net.DialUDP("udp", nil, groupAddr)
// 	if err != nil {
// 		log.Fatalf("Failed to dial UDP: %v", err)
// 	}

// 	pubsub := mpubsub.New(rc, wc)

// 	pubsub.Subscribe(&DemoServer{})

// 	go pubsub.Listen()

// 	time.Sleep(1 * time.Second)

// 	msg := &protocol.PeerAnnouncementMessage{
// 		NodeID:         *oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXT"),
// 		Address:        "127.0.0.1:5001",
// 		SequenceNumber: 0,
// 	}
// 	// msg := &protocol.PeerSyncRequest{
// 	// 	NodeID:         *oid.FromStringMustParse("AGVABZGP4JKRP2AK4YSGHKS7WH2C7P44XDNWJ4EDYP4B2GQ5UFQQQAXT"),
// 	// 	SequenceNumber: 0,
// 	// 	BatchSize:      10,
// 	// }

// 	err = pubsub.Publish("DemoServer.PeerAnnouncement", msg)

// 	time.Sleep(1 * time.Second)

// 	return

// 	// testString := "/Anime/Boku no Pico/[RAW]/Boku no Pico - 01 (v2) [720p].mkv"
// 	// sha256Hash := sha256.Sum256([]byte(testString))

// 	// o, err := oid.Encode(oid.OidTypeRawBlock, sha256Hash)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to encode OID: %v", err)
// 	// }

// 	// log.Infof("Encoded OID: %s", o.String())

// 	// o2, err := oid.FromString(o.String())
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to decode OID: %v", err)
// 	// }

// 	// log.Infof("Decoded OID: %s, type: %d", o2.String(), o2.Type())

// 	// b, err := cbor.Marshal(o)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to encode OID to CBOR: %v", err)
// 	// }
// 	// log.Infof("CBOR encoded OID: %s", hex.EncodeToString(b))

// 	// o3 := &oid.Oid{}
// 	// if err := cbor.Unmarshal(b, o3); err != nil {
// 	// 	log.Fatalf("Failed to decode CBOR OID: %v", err)
// 	// }
// 	// log.Infof("Decoded CBOR OID: %s, type: %d", o3.String(), o3.Type())

// 	// blk := &block.Block{
// 	// 	Oid:    *o,
// 	// 	Length: uint64(len(testString)),
// 	// }

// 	// b, err = cbor.Marshal(blk)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to encode Block to CBOR: %v", err)
// 	// }
// 	// log.Infof("CBOR encoded Block: %s", hex.EncodeToString(b))

// 	nodeID, err := oid.Encode(oid.OidTypeNode, sha256.Sum256([]byte("TestNode")))
// 	if err != nil {
// 		log.Fatalf("Failed to encode OID: %v", err)
// 	}
// 	log.Infof("NodeID: %s", nodeID.String())

// 	srv := crpc.NewServer()
// 	srv.Register(&DemoServer{})
// 	l, err := net.Listen("tcp", ":5001")
// 	if err != nil {
// 		log.Fatalf("Failed to create a listener: %v", err)
// 	}
// 	go srv.Serve(l)

// 	time.Sleep(1 * time.Second)

// 	cli, err := crpc.Dial("tcp", "localhost:5001")
// 	if err != nil {
// 		log.Fatalf("Failed to create RPC client: %v", err)
// 	}

// 	req := &protocol.PeerSyncRequest{
// 		NodeID:         *nodeID,
// 		SequenceNumber: 0,
// 		BatchSize:      10,
// 	}
// 	res := &protocol.PeerSyncResponse{}

// 	err = cli.Call(ctx, "DemoServer.PeerSync", req, res)
// 	if err != nil {
// 		log.Errorf("Failed to call PeerSync: %v", err)
// 	}
// 	for _, e := range res.Entries {
// 		log.Infof("Entry: seq=%d, oid=%s, length=%d, time=%s, deleted=%t", e.Sequence, e.Metadata.Oid.String(), e.Metadata.Length, e.Metadata.UpdateTime, e.Metadata.IsDeleted)
// 	}

// 	err = cli.Call(ctx, "DemoServer.PeerSync", req, res)
// 	if err != nil {
// 		log.Errorf("Failed to call PeerSync: %v", err)
// 	}
// 	for _, e := range res.Entries {
// 		log.Infof("Entry: seq=%d, oid=%s, length=%d, time=%s, deleted=%t", e.Sequence, e.Metadata.Oid.String(), e.Metadata.Length, e.Metadata.UpdateTime, e.Metadata.IsDeleted)
// 	}

// 	// srv := server.NewServer(*nodeID, &blockIndex{})
// 	// srv.Serve(ctx)

// 	// time.Sleep(1 * time.Second)

// 	// client, err := client.Dial("localhost:5001")
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to create RPC client: %v", err)
// 	// }

// 	// req := &protocol.PeerSyncRequest{
// 	// 	NodeID:         *nodeID,
// 	// 	SequenceNumber: 0,
// 	// 	BatchSize:      10,
// 	// }

// 	// cctx, _ := context.WithTimeout(ctx, 1*time.Second)
// 	// res, err := client.PeerSync(cctx, req)
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to call PeerSync: %v", err)
// 	// }

// 	// log.Infof("Response from %s, entries: %d", res.NodeID.String(), len(res.Entries))
// 	// for _, e := range res.Entries {
// 	// 	log.Infof("Entry: seq=%d, oid=%s, length=%d, time=%s, deleted=%t", e.Sequence, e.Metadata.Oid.String(), e.Metadata.Length, e.Metadata.UpdateTime,
// 	// 		e.Metadata.IsDeleted)
// 	// }
// }

func registerGlobalFlags(fset *flag.FlagSet) {
	flag.VisitAll(func(f *flag.Flag) {
		fset.Var(f.Value, f.Name, f.Usage)
	})
}

func checkConfig(cfg string) {
	if cfg == "" {
		log.Fatal("Config file not specified")
	}
}

// main is the entry point of the application.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	configFile := flag.String("config", "", "Path to config file")
	logLevel := flag.String("loglevel", "debug", "Log level")

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	registerGlobalFlags(initCmd)

	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	mountPoint := serveCmd.String("mountpoint", "/tmp/juren-cluster/mount", "Path to mount the filesystem at")
	registerGlobalFlags(serveCmd)

	publishCmd := flag.NewFlagSet("publish", flag.ExitOnError)
	registerGlobalFlags(publishCmd)

	testCmd := flag.NewFlagSet("test", flag.ExitOnError)
	registerGlobalFlags(testCmd)

	infoCmd := flag.NewFlagSet("info", flag.ExitOnError)
	registerGlobalFlags(infoCmd)

	if len(os.Args) < 2 {
		log.WithField("args", os.Args).Fatal("Expected a subcommand")
	}
	cmd, args := os.Args[1], os.Args[2:]

	switch cmd {
	case "init":
		initCmd.Parse(args)
		checkConfig(*configFile)
		setLogLevel(*logLevel)
		cfg := config.NewEmptyConfig(*configFile)
		commands.RunInit(ctx, cfg)
	case "serve":
		serveCmd.Parse(args)
		checkConfig(*configFile)
		setLogLevel(*logLevel)
		cfg, err := config.NewConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		commands.RunServe(ctx, cfg, *mountPoint)
	case "publish":
		publishCmd.Parse(args)
		checkConfig(*configFile)
		setLogLevel(*logLevel)
		cfg, err := config.NewConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		commands.RunPublish(ctx, cfg)
	case "test":
		testCmd.Parse(args)
		checkConfig(*configFile)
		setLogLevel(*logLevel)
		cfg, err := config.NewConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		commands.RunTest(ctx, cfg)
	case "info":
		infoCmd.Parse(args)
		checkConfig(*configFile)
		setLogLevel(*logLevel)
		cfg, err := config.NewConfigFromFile(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		commands.RunInfo(ctx, cfg)
	default:
		log.Fatalf("Invalid subcommand '%s'", os.Args[1])
	}
}
