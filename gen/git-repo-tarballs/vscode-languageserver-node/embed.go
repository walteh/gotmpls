package vscodelanguageservernode

import _ "embed"

//go:embed vscode-languageserver-node.tar.gz
var Data []byte
var Ref string = "tags/release/jsonrpc/9.0.0-next.6"
