package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	vscodelanguageservernode "github.com/walteh/gotmpls/gen/git-repo-tarballs/vscode-languageserver-node"
	"github.com/walteh/gotmpls/pkg/archive"
)

//go:generate go run . -o ../
func processinline() {

	dir := build_tmp_dir()
	defer os.RemoveAll(dir)

	model := parse(filepath.Join(dir, "protocol/metaModel.json"))

	findTypeNames(model)
	generateOutput(model)

	fileHdr = fileHeader2(model)

	// write the files
	writemyclient()
	writemyserver()
	writeprotocol()
	writejsons()

	checkTables()
}

// create the common file header for the output files
func fileHeader2(model *Model) string {

	format := `// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code generated for LSP. DO NOT EDIT.

package protocol

// Code generated from %[1]s at ref %[2]s (hash %[3]s).
// %[4]s/blob/%[2]s/%[1]s
// LSP metaData.version = %[5]s.

`
	return fmt.Sprintf(format,
		"protocol/metaModel.json",    // 1
		lspGitRef,                    // 2
		vscodelanguageservernode.Ref, // 3
		vscodeRepo,                   // 4
		model.Version.Version)        // 5
}

func build_tmp_dir() string {

	repo := vscodelanguageservernode.Data
	tmpDir, err := os.MkdirTemp("", "vscode-languageserver-node")
	if err != nil {
		panic(err)
	}

	err = archive.ExtractTarGz(repo, tmpDir)
	if err != nil {
		panic(err)
	}

	// list files in dir
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		panic(err)
	}
	dir := ""
	for _, file := range files {
		if file.IsDir() {
			dir = file.Name()
			break
		}
	}

	refNoTags := strings.TrimPrefix(vscodelanguageservernode.Ref, "tags/")

	refd := []string{tmpDir, dir, ".git"}
	refd1 := append(refd, "HEAD")
	refd2 := append(refd, refNoTags)

	refFile1 := filepath.Join(refd1...)
	refFile2 := filepath.Join(refd2...)

	//mkdir
	err = os.MkdirAll(filepath.Dir(refFile1), 0755)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(filepath.Dir(refFile2), 0755)
	if err != nil {
		panic(err)
	}

	// write ref to file
	err = os.WriteFile(refFile1, []byte("ref: "+refNoTags), 0644)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(refFile2, []byte(refNoTags), 0644)
	if err != nil {
		panic(err)
	}

	// cmd := exec.Command("task", "tool:generate", "--", "-d", tmpDir+"/"+dir, "-o", "./out/lsp")
	// cmd.Dir = "../../.."
	// out, err := cmd.CombinedOutput()
	// fmt.Println(string(out))
	// if err != nil {
	// 	panic(err)
	// }

	return filepath.Join(tmpDir, dir)
}

func writemyclient() {
	out := new(bytes.Buffer)
	fmt.Fprintln(out, fileHdr)
	out.WriteString(
		`import (
	"context"

	"github.com/creachadair/jrpc2/handler"
)
`)
	out.WriteString("type Client interface {\n")
	for _, k := range cdecls.keys() {
		out.WriteString(cdecls[k])
	}
	out.WriteString("}\n\n")
	out.WriteString(`func buildClientDispatchMap(client Client) handler.Map {
	return handler.Map{
`)
	for _, k := range ccases.keys() {
		out.WriteString(ccases[k])
	}
	out.WriteString(("\t}\n}\n\n"))
	for _, k := range cfuncs.keys() {
		out.WriteString(cfuncs[k])
	}
	formatTo("tsclient.go", out.Bytes())
}

func writemyserver() {
	out := new(bytes.Buffer)
	fmt.Fprintln(out, fileHdr)
	out.WriteString(
		`import (
	"context"

	"github.com/creachadair/jrpc2/handler"
)
`)
	out.WriteString("type Server interface {\n")
	for _, k := range sdecls.keys() {
		out.WriteString(sdecls[k])
	}
	out.WriteString(`
}

func buildServerDispatchMap(server Server) handler.Map {
	return handler.Map{
`)
	for _, k := range scases.keys() {
		out.WriteString(scases[k])
	}
	out.WriteString(("\t}\n}\n\n"))
	for _, k := range sfuncs.keys() {
		out.WriteString(sfuncs[k])
	}
	formatTo("tsserver.go", out.Bytes())
}

func genCase(_ *Model, method string, param, result *Type, dir string) {
	out := new(bytes.Buffer)
	fmt.Fprintf(out, "\t%q:", method)
	// var p string
	fname := methodName(method)

	hasParams := notNil(param)
	hasResult := notNil(result)
	if hasParams && hasResult {
		fmt.Fprintf(out, "createHandler(%%s.%s),", fname)
	} else if hasParams {
		fmt.Fprintf(out, "createEmptyResultHandler(%%s.%s),", fname)
	} else if hasResult {
		fmt.Fprintf(out, "createEmptyParamsHandler(%%s.%s),", fname)
	} else {
		fmt.Fprintf(out, "createEmptyHandler(%%s.%s),", fname)
	}
	out.WriteString("\n")
	msg := out.String()
	switch dir {
	case "clientToServer":
		scases[method] = fmt.Sprintf(msg, "server")
	case "serverToClient":
		ccases[method] = fmt.Sprintf(msg, "client")
	case "both":
		scases[method] = fmt.Sprintf(msg, "server")
		ccases[method] = fmt.Sprintf(msg, "client")
	default:
		log.Fatalf("impossible direction %q", dir)
	}
}

func genFunc(_ *Model, method string, param, result *Type, dir string, isnotify bool) {
	out := new(bytes.Buffer)
	var p, r string
	var goResult string
	if notNil(param) {
		p = ", params *" + goplsName(param)
	}
	if notNil(result) {
		goResult = goplsName(result)
		if !hasNilValue(goResult) {
			goResult = "*" + goResult
		}
		r = fmt.Sprintf("(%s, error)", goResult)
	} else {
		r = "error"
	}
	// special gopls compatibility case
	switch method {
	case "workspace/configuration":
		// was And_Param_workspace_configuration, but the type substitution doesn't work,
		// as ParamConfiguration is embedded in And_Param_workspace_configuration
		p = ", params *ParamConfiguration"
		r = "([]LSPAny, error)"
		goResult = "[]LSPAny"
	}
	fname := methodName(method)
	fmt.Fprintf(out, "func (s *Callback%%s) %s(ctx context.Context%s) %s {\n", fname, p, r)

	if !notNil(result) {
		if isnotify {
			if notNil(param) {
				fmt.Fprintf(out, "\treturn createNotify(ctx, s, %q, params)\n", method)
			} else {
				fmt.Fprintf(out, "\treturn createEmptyNotify(ctx, s, %q)\n", method)
			}
		} else {
			if notNil(param) {
				fmt.Fprintf(out, "\treturn createEmptyResultCallback(ctx, s, %q, params)\n", method)
			} else {
				fmt.Fprintf(out, "\treturn createEmptyCallback(ctx, s, %q)\n", method)
			}
		}
	} else {
		fmt.Fprintf(out, "\tvar result %s\n", goResult)
		if isnotify {
			if notNil(param) {
				fmt.Fprintf(out, "\treturn createNotify(ctx, s, %q, params)\n", method)
			} else {
				fmt.Fprintf(out, "\t\tif err := createEmptyNotify(ctx, s, %q); err != nil {\n", method)
			}
		} else {
			if notNil(param) {
				fmt.Fprintf(out, "\t\tif err := createCallback(ctx, s, %q, params, &result); err != nil {\n", method)
			} else {
				fmt.Fprintf(out, "\t\tif err := createEmptyParamsCallback(ctx, s, %q, &result); err != nil {\n", method)
			}
		}
		fmt.Fprintf(out, "\t\treturn nil, err\n\t}\n\treturn result, nil\n")
	}
	out.WriteString("}\n")
	msg := out.String()
	switch dir {
	case "clientToServer":
		sfuncs[method] = fmt.Sprintf(msg, "Server")
	case "serverToClient":
		cfuncs[method] = fmt.Sprintf(msg, "Client")
	case "both":
		sfuncs[method] = fmt.Sprintf(msg, "Server")
		cfuncs[method] = fmt.Sprintf(msg, "Client")
	default:
		log.Fatalf("impossible direction %q", dir)
	}
}
