package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"gen/pb/rpc/tools/rpcc"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
)

var (
	genBaseDir = filepath.Join(os.Getenv("GOPATH"), "gen", "rpc")
	outDir     = flag.String("out_dir", "", "Output dir relative to $GOPATH/gen/rpc")
)

func main() {
	flag.Parse()
	logger := xlog.NewConsoleLogger()

	const usage = "usage: rpcc [--out_dir=<OUT_DIR>] service_def_file"
	if len(flag.Args()) < 1 {
		logger.Fatal(usage)
	}

	var err error
	text, err := ioutil.ReadFile(flag.Args()[0])
	if err != nil {
		logger.Fatalf("Failed to read %s: ", err)
	}
	svcDef := &rpcc_proto.Service{}
	if err = proto.UnmarshalText(string(text), svcDef); err != nil {
		logger.Fatalf("Failed to unmarshal service definition: %s", err)
	}
	out, err := os.Create(filepath.Join(genBaseDir, *outDir, "rpc_def.go"))
	if err != nil {
		logger.Fatalf("Failed to create file: %s", err)
	}
	defer out.Close()
	t := template.Must(template.New("serviceDef").Parse(tmpl))
	if err = t.Execute(out, svcDef); err != nil {
		logger.Fatalf("Failed to execute template: %s", err)
	}
}
