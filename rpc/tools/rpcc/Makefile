protos:
	cd proto && mkdir -p gen-go && protoc --proto_path=./ --go_out=./gen-go *.proto

build: protos
	go build -o ./rpcc
