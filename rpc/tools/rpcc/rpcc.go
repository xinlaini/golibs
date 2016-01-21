package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/golang/protobuf/proto"
	xlog "github.com/xinlaini/golibs/log"
	"github.com/xinlaini/golibs/rpc/tools/rpcc/proto/gen-go"
)

var (
	outDir = flag.String("out_dir", "./", "output directory")
)

func main() {
	flag.Parse()

	const usage = "usage: rpcc [--out_dir=<OUT_DIR>] service_def_file"
	if len(flag.Args()) < 1 {
		xlog.Fatal(usage)
	}

	var err error
	text, err := ioutil.ReadFile(flag.Args()[0])
	if err != nil {
		xlog.Fatalf("Failed to read %s: ", err)
	}
	svcDef := &rpcc_proto.Service{}
	if err = proto.UnmarshalText(string(text), svcDef); err != nil {
		xlog.Fatalf("Failed to unmarshal service definition: %s", err)
	}
	out, err := os.Create(filepath.Join(*outDir, "rpc_def.go"))
	if err != nil {
		xlog.Fatalf("Failed to create file: %s", err)
	}
	defer out.Close()
	t := template.Must(template.New("serviceDef").Parse(tmpl))
	if err = t.Execute(out, svcDef); err != nil {
		xlog.Fatalf("Failed to execute template: %s", err)
	}
}
