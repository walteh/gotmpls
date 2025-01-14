package nvimlspconfig
import _ "embed"
//go:embed nvim-lspconfig.tar.gz
var Data []byte
var Ref string = "tags/v1.3.0"
