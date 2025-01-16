package nvim

const sharedNeovimConfig = `
-- Enable debug logging
vim.lsp.set_log_level("debug")

local lspconfig = require 'lspconfig'
local configs = require 'lspconfig.configs'
local util = require 'lspconfig.util'
local async = require 'lspconfig.async'

-- Print loaded configs for debugging
print("Available LSP configs:", vim.inspect(configs))

-- Configure capabilities
local capabilities = vim.lsp.protocol.make_client_capabilities()
capabilities.textDocument.hover = {
    dynamicRegistration = true,
    contentFormat = { "plaintext", "markdown" }
}

-- Use an on_attach function to only map the following keys
local on_attach = function(client, bufnr)
    print("LSP client attached:", vim.inspect(client))
    print("Buffer:", bufnr)
    print("Client capabilities:", vim.inspect(client.server_capabilities))
end
`

type NeovimConfig interface {
	DefaultConfig(socketPath string) string
	DefaultSetup() string
}

type GoTemplateConfig struct{}

func (c *GoTemplateConfig) DefaultConfig(socketPath string) string {
	return `if not configs.go_template then
configs.go_template = {
        default_config = {
            cmd = { 'go', 'run', 'github.com/walteh/go-tmpl-typer/cmd/stdio-proxy', '` + socketPath + `' },
            filetypes = { 'go-template', 'gotmpl' },
            root_dir = function(fname)
                local path = vim.fn.getcwd()
                print("Setting root dir to:", path)
                return path
            end,
            init_options = {
                usePlaceholders = true,
                completeUnimported = true,
                staticcheck = true
            },
            settings = { },
            flags = {
                debounce_text_changes = 0,
                allow_incremental_sync = true,
            },
			single_file_support = true,
        },
    }
    -- Set up immediately after defining
    lspconfig.go_template.setup {}
    print("go_template server setup complete")
end`
}

func (c *GoTemplateConfig) DefaultSetup() string {
	return `if not lspconfig.go_template then
    print("ERROR: go_template config not found!")
end`
}

type GoplsConfig struct{}

func (c *GoplsConfig) DefaultConfig(socketPath string) string {
	return `if not configs.gopls then
		print("Setting up gopls server config")
	configs.gopls = {
		default_config = {
			cmd = { 'go', 'run', 'github.com/walteh/go-tmpl-typer/cmd/stdio-proxy', '` + socketPath + `' },
			filetypes = { 'go', 'gomod', 'gowork', 'gotmpl' },
            root_dir = function(fname)
                local path = vim.fn.getcwd()
                print("Setting root dir to:", path)
                return path
            end,
			single_file_support = true,
			flags = {
				debounce_text_changes = 0,
				allow_incremental_sync = true,
			},
		}
	}
	-- Set up immediately after defining
	lspconfig.gopls.setup {}
	print("gopls server setup complete")
end`
}

func (c *GoplsConfig) DefaultSetup() string {
	return `if not lspconfig.gopls then
		print("ERROR: gopls config not found!")
	end`
}
