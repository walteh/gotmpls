package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	vscodelanguageservernode "github.com/walteh/go-tmpl-typer/gen/git-repo-tarballs/vscode-languageserver-node"
	"github.com/walteh/go-tmpl-typer/pkg/archive"
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
	writeclient()
	writeserver()
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
