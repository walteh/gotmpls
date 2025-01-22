//go:build tools

package main

import (
	_ "cmd/test2json" //export

	_ "gotest.tools/gotestsum" //export

	_ "github.com/google/go-containerregistry/cmd/crane" //export

	_ "connectrpc.com/connect/cmd/protoc-gen-connect-go"
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/bufbuild/plugins/cmd/download-plugins"
	_ "github.com/bufbuild/plugins/cmd/latest-plugins"
	_ "github.com/go-task/task/v3/cmd/task"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/goreleaser/goreleaser/v2"
	_ "github.com/oligot/go-mod-upgrade"
	_ "github.com/srikrsna/protoc-gen-gotag"
	_ "github.com/vektra/mockery/v2"
	_ "github.com/walteh/retab/v2/cmd/retab"

	_ "github.com/alecthomas/participle/v2/cmd/railroad"
	_ "github.com/ianlewis/todos/internal/cmd/todos"
	_ "github.com/walteh/copyrc/cmd/copyrc"
	_ "github.com/walteh/gotmpls/gen/participle"
)
