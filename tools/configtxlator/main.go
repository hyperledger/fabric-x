/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/cockroachdb/errors"
	"github.com/gorilla/handlers"
	"github.com/hyperledger/fabric-lib-go/common/flogging"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	_ "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	_ "github.com/hyperledger/fabric-protos-go-apiv2/orderer"
	_ "github.com/hyperledger/fabric-protos-go-apiv2/orderer/etcdraft"
	_ "github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/hyperledger/fabric-x-common/common/metadata"
	"github.com/hyperledger/fabric-x-common/protolator"
	"github.com/hyperledger/fabric-x-common/tools/configtxlator/rest"
	"github.com/hyperledger/fabric-x-common/tools/configtxlator/update"
)

const programName = "configtxlator"

var (
	commitSHA = metadata.CommitSHA
	version   = metadata.Version
)

// command line flags.
var (
	app = kingpin.New("configtxlator", "Utility for generating Hyperledger Fabric channel configurations")

	start    = app.Command("start", "Start the configtxlator REST server")
	hostname = start.Flag(
		"hostname",
		"The hostname or IP on which the REST server will listen",
	).Default("0.0.0.0").String()
	port = start.Flag("port", "The port on which the REST server will listen").Default("7059").Int()
	cors = start.Flag("CORS", "Allowable CORS domains, e.g. '*' or 'www.example.com' (may be repeated).").Strings()

	protoEncode     = app.Command("proto_encode", "Converts a JSON document to protobuf.")
	protoEncodeType = protoEncode.Flag(
		"type",
		"The type of protobuf structure to encode to.  For example, 'common.Config'.",
	).Required().String()
	protoEncodeSource = protoEncode.Flag("input", "A file containing the JSON document.").Default(os.Stdin.Name()).File()
	protoEncodeDest   = protoEncode.Flag(
		"output",
		"A file to write the output to.",
	).Default(os.Stdout.Name()).OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)

	protoDecode     = app.Command("proto_decode", "Converts a proto message to JSON.")
	protoDecodeType = protoDecode.Flag(
		"type",
		"The type of protobuf structure to decode from.  For example, 'common.Config'.",
	).Required().String()
	protoDecodeSource = protoDecode.Flag("input", "A file containing the proto message.").Default(os.Stdin.Name()).File()
	protoDecodeDest   = protoDecode.Flag(
		"output",
		"A file to write the JSON document to.",
	).Default(os.Stdout.Name()).OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)

	computeUpdate          = app.Command("compute_update", "Takes two marshaled common.Config messages and computes the config update which transitions between the two.")
	computeUpdateOriginal  = computeUpdate.Flag("original", "The original config message.").File()
	computeUpdateUpdated   = computeUpdate.Flag("updated", "The updated config message.").File()
	computeUpdateChannelID = computeUpdate.Flag(
		"channel_id",
		"The name of the channel for this update.",
	).Required().String()
	computeUpdateDest = computeUpdate.Flag(
		"output",
		"A file to write the JSON document to.",
	).Default(os.Stdout.Name()).OpenFile(os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)

	versionCmd = app.Command("version", "Show version information")
)

var logger = flogging.MustGetLogger("configtxlator")

func main() {
	kingpin.Version("0.0.1")
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	// "start" command
	case start.FullCommand():
		startServer(fmt.Sprintf("%s:%d", *hostname, *port), *cors)
	// "proto_encode" command
	case protoEncode.FullCommand():
		defer func() {
			if err := (*protoEncodeSource).Close(); err != nil {
				logger.Warnf("error closing protoEncodeSource: %s", err)
			}
		}()
		defer func() {
			if err := (*protoEncodeDest).Close(); err != nil {
				logger.Warnf("error closing protoEncodeDest: %s", err)
			}
		}()
		err := encodeProto(*protoEncodeType, *protoEncodeSource, *protoEncodeDest)
		if err != nil {
			app.Fatalf("Error decoding: %s", err)
		}
	case protoDecode.FullCommand():
		defer func() {
			if err := (*protoDecodeSource).Close(); err != nil {
				logger.Warnf("error closing protoDecodeSource: %s", err)
			}
		}()
		defer func() {
			if err := (*protoDecodeDest).Close(); err != nil {
				logger.Warnf("error closing protoDecodeDest: %s", err)
			}
		}()
		err := decodeProto(*protoDecodeType, *protoDecodeSource, *protoDecodeDest)
		if err != nil {
			app.Fatalf("Error decoding: %s", err)
		}
	case computeUpdate.FullCommand():
		defer func() {
			if err := (*computeUpdateOriginal).Close(); err != nil {
				logger.Warnf("error closing computeUpdateOriginal: %s", err)
			}
		}()
		defer func() {
			if err := (*computeUpdateUpdated).Close(); err != nil {
				logger.Warnf("error closing computeUpdateUpdated: %s", err)
			}
		}()
		defer func() {
			if err := (*computeUpdateDest).Close(); err != nil {
				logger.Warnf("error closing computeUpdateDest: %s", err)
			}
		}()
		err := computeUpdt(*computeUpdateOriginal, *computeUpdateUpdated, *computeUpdateDest, *computeUpdateChannelID)
		if err != nil {
			app.Fatalf("Error computing update: %s", err)
		}
	// "version" command
	case versionCmd.FullCommand():
		fmt.Println(getVersionInfo())
	}
}

func startServer(address string, cors []string) {
	var err error

	listener, err := net.Listen("tcp", address)
	if err != nil {
		app.Fatalf("Could not bind to address '%s': %s", address, err)
	}

	server := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if len(cors) > 0 {
		origins := handlers.AllowedOrigins(cors)
		methods := handlers.AllowedMethods([]string{http.MethodPost})
		headers := handlers.AllowedHeaders([]string{"Content-Type"})
		logger.Infof("Serving HTTP requests on %s with CORS %v", listener.Addr(), cors)
		server.Handler = handlers.CORS(origins, methods, headers)(rest.NewRouter())
	} else {
		logger.Infof("Serving HTTP requests on %s", listener.Addr())
		server.Handler = rest.NewRouter()
	}

	err = server.Serve(listener)
	app.Fatalf("Error starting server:[%s]\n", err)
}

func encodeProto(msgName string, input, output *os.File) error {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(msgName))
	if err != nil {
		return errors.Wrapf(err, "error encode input")
	}

	msgType := reflect.TypeOf(mt.Zero().Interface())

	if msgType == nil {
		return errors.Errorf("message of type %s unknown", msgType)
	}
	msg := reflect.New(msgType.Elem()).Interface().(proto.Message)

	err = protolator.DeepUnmarshalJSON(input, msg)
	if err != nil {
		return errors.Wrapf(err, "error decoding input")
	}

	if msg == nil {
		return errors.New("error marshaling: proto: Marshal called with nil")
	}
	out, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrapf(err, "error marshaling")
	}

	_, err = output.Write(out)
	if err != nil {
		return errors.Wrapf(err, "error writing output")
	}

	return nil
}

func decodeProto(msgName string, input, output *os.File) error {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(msgName))
	if err != nil {
		return errors.Wrapf(err, "error encode input")
	}

	msgType := reflect.TypeOf(mt.Zero().Interface())

	if msgType == nil {
		return errors.Errorf("message of type %s unknown", msgType)
	}
	msg := reflect.New(msgType.Elem()).Interface().(proto.Message)

	in, err := io.ReadAll(input)
	if err != nil {
		return errors.Wrapf(err, "error reading input")
	}

	err = proto.Unmarshal(in, msg)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling")
	}

	err = protolator.DeepMarshalJSON(output, msg)
	if err != nil {
		return errors.Wrapf(err, "error encoding output")
	}

	return nil
}

func computeUpdt(original, updated, output *os.File, channelID string) error {
	origIn, err := io.ReadAll(original)
	if err != nil {
		return errors.Wrapf(err, "error reading original config")
	}

	origConf := &cb.Config{}
	err = proto.Unmarshal(origIn, origConf)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling original config")
	}

	updtIn, err := io.ReadAll(updated)
	if err != nil {
		return errors.Wrapf(err, "error reading updated config")
	}

	updtConf := &cb.Config{}
	err = proto.Unmarshal(updtIn, updtConf)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling updated config")
	}

	cu, err := update.Compute(origConf, updtConf)
	if err != nil {
		return errors.Wrapf(err, "error computing config update")
	}

	if cu == nil {
		return errors.New("error marshaling computed config update: proto: Marshal called with nil")
	}

	cu.ChannelId = channelID

	outBytes, err := proto.Marshal(cu)
	if err != nil {
		return errors.Wrapf(err, "error marshaling computed config update")
	}

	_, err = output.Write(outBytes)
	if err != nil {
		return errors.Wrapf(err, "error writing config update to output")
	}

	return nil
}

func getVersionInfo() string {
	return fmt.Sprintf("%s:\n Version: %s\n Commit SHA: %s\n Go version: %s\n OS/Arch: %s",
		programName, version, commitSHA, runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}
