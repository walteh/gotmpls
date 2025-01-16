package nvim

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
           -- on_attach = on_attach,
            init_options = {
                usePlaceholders = true,
                completeUnimported = true,
                staticcheck = true
            },
            settings = { },
            flags = {
                debounce_text_changes = 0,  -- Disable debouncing
                allow_incremental_sync = false,  -- Disable incremental sync
                server_side_fuzzy_completion = false,  -- Disable fuzzy completion
            }
        },
    }
end`
}

func (c *GoTemplateConfig) DefaultSetup() string {
	return `if lspconfig.go_template then
    print("Setting up go_template server")
    lspconfig.go_template.setup {
--        on_attach = function(client, bufnr)
--            print("go_template server attached to buffer", bufnr)
--            print("Client capabilities:", vim.inspect(client.server_capabilities))
--			vim.api.nvim_buf_set_keymap(bufnr, 'n', 'K', '<cmd>lua vim.lsp.buf.hover()<CR>', { noremap = true, silent = false })
--            on_attach(client, bufnr)
--        end,
        flags = {
            debounce_text_changes = 0,  -- Disable debouncing
            allow_incremental_sync = false,  -- Disable incremental sync
            server_side_fuzzy_completion = false,  -- Disable fuzzy completion
        }
    }
    print("go_template server setup complete")
else
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
-- 			on_attach = on_attach,
            root_dir = function(fname)
                local path = vim.fn.getcwd()
                print("Setting root dir to:", path)
                return path
            end,
			single_file_support = true,
		}
	}
	print("gopls server config complete")
else
	print("WARNING: gopls config found!", configs.gopls)
end`
}

func (c *GoplsConfig) DefaultSetup() string {
	return `
	if lspconfig.gopls then
		print("Setting up gopls server")
		lspconfig.gopls.setup {
--			on_attach = function(client, bufnr)
--				print("gopls server attached to buffer", bufnr)
--				print("Client capabilities:", vim.inspect(client.server_capabilities))
--				vim.api.nvim_buf_set_keymap(bufnr, 'n', 'K', '<cmd>lua vim.lsp.buf.hover()<CR>', { noremap = true, silent = false })
--				on_attach(client, bufnr)
--			end,
		}
		print("gopls server setup complete")
	else
		print("ERROR: gopls config not found!")
	end`
}
