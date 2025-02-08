import * as pluginJs from "@eslint/js";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import * as tsParser from "@typescript-eslint/parser";
import type { Linter } from "eslint";
import * as prettierPlugin from "eslint-plugin-prettier";
import simpleImportSort from "eslint-plugin-simple-import-sort";
import * as globals from "globals";

export default [
	{
		files: ["**/*.ts"],
	},
	{
		languageOptions: {
			globals: {
				node: true,
			},
			parser: tsParser,
			parserOptions: {
				project: ["./tsconfig.json"],
				tsconfigRootDir: ".",
				sourceType: "module",
				ecmaVersion: "latest",
			},
		},
	},
	{
		plugins: tsPlugin,
	},
	{
		plugins: {
			"simple-import-sort": simpleImportSort,
		},
		rules: {
			"simple-import-sort/imports": [
				"error",
				{
					groups: [
						// Side effect imports.
						["^\\u0000"],
						// bun builtins prefixed with `bun:`.
						["^bun:"],
						// node builtins prefixed with `node:`.
						["^node:"],
						// vscode builtins prefixed with `vscode:`.
						["^vscode:"],
						// Packages.
						["^@?\\w"],
						// Absolute imports and other imports such as Vue-style `@/foo`.
						["^"],
						// Relative imports.
						["^\\."],
					],
				},
			],
			"simple-import-sort/exports": "error",
		},
	},

	{
		plugins: {
			prettier: prettierPlugin,
		},
		// rules: {
		// 	"prettier/prettier": "error",
		// },
	},
] satisfies Linter.Config[];
