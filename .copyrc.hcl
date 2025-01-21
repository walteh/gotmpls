copy {
	source {
		repo = "github.com/golang/tools"
		ref  = "master"
		path = "gopls/internal/protocol/generate"
	}
	destination {
		path = "./pkg/lsp/protocol/generator"
	}
	options {
		replacements = [
			{
				old = "func processinline()",
				new = "func processinline_old()",
			},
			{
				old = "golang.org/x/tools/internal/jsonrpc2",
				new = "github.com/creachadair/jrpc2"
			},
			{
				old = "reply jsonrpc2.Replier, r jsonrpc2.Request",
				new = "conn *jrpc2.Server, r *jrpc2.Request"
			},
			{
				old = "func genCase(",
				new = "func genCase_old("
			},
			{
				old = "func genFunc(",
				new = "func genFunc_old("
			},
			{
				old = "UnmarshalJSON(r.Params(), &params)",
				new = "UnmarshalJSON(r, &params)"
			},
			{
				old = "reply(ctx, ",
				new = "reply_fwd(ctx, conn, r,"
			},
			{
				old = "sendParseError(ctx, reply,",
				new = "sendParseError(ctx, conn, r,"
			},
			{
				old = "recoverHandlerPanic(r.Method())",
				new = "recoverHandlerPanic(r.Method)"
			},
			{
				old = "tsprotocol.go",
				new = "tsprotocol.gen.go"
			},
			{
				old = "tsserver.go",
				new = "tsserver.gen.go"
			},
			{
				old = "tsclient.go",
				new = "tsclient.gen.go"
			},
			{
				old = "tsjson.go",
				new = "tsjson.gen.go"
			},
		]
		ignore_files = [
			"*.txt",
		]
	}
}

archive {
	source {
		repo = "github.com/neovim/nvim-lspconfig"
		ref  = "tags/v1.3.0"
	}
	destination {
		path = "./gen/git-repo-tarballs"
	}
	options {
		go_embed = true
	}
}

archive {
	source {
		repo = "github.com/microsoft/vscode-languageserver-node"
		ref  = "tags/release/jsonrpc/9.0.0-next.6"
	}
	destination {
		path = "./gen/git-repo-tarballs"
	}
	options {
		go_embed = true
	}
}

