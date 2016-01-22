package main

const tmpl = `package {{.Package}}

import "reflect"
{{range .Import}}import{{if .As}} {{.As}}{{end}} "{{.Path}}"
{{end}}import "github.com/xinlaini/golibs/rpc"

type {{.Name}}Service interface {
{{range .Method}}	{{.Name}}(ctx *rpc.ServerContext, req *{{.RequestProto}}) (*{{.ResponseProto}}, error)
{{end}}}

type {{.Name}}Client struct {
	rpcClient *rpc.Client
}

func New{{.Name}}Client(ctrl *rpc.Controller, opts rpc.ClientOptions) (*{{.Name}}Client, error) {
	rpcClient, err := ctrl.NewClient(opts)
	if err != nil {
		return nil, err
	}
	return &{{.Name}}Client{rpcClient: rpcClient}, nil
}
{{with $root := .}}{{range .Method}}
func (c *{{$root.Name}}Client) {{.Name}}(ctx *rpc.ClientContext, req *{{.RequestProto}}) (*{{.ResponseProto}}, err) {
	pbResp, err := c.ctrl.Call("{{.Name}}", ctx, req, reflect.TypeOf((*{{.ResponseProto}})(nil)).Elem())
	if err != nil {
		return nil, err
	}
	if pbResp == nil {
		return nil, nil
	}
	return pbResp.(*{{.ResponseProto}}), nil
}
{{end}}{{end}}
`
