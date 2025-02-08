import tsPlugin from "@typescript-eslint/eslint-plugin";
import * as tsParser from "@typescript-eslint/parser";
import type { Linter } from "eslint";
import functional from "eslint-plugin-functional";
import prettierPlugin from "eslint-plugin-prettier";
import securityPlugin from "eslint-plugin-security";
import simpleImportSort from "eslint-plugin-simple-import-sort";
import importPlugin from "eslint-plugin-unused-imports";

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
			security: securityPlugin,
		},
	},
	// @ts-expect-error
	(await import("eslint-plugin-functional")).default.configs.all,
	{
		plugins: {
			"simple-import-sort": simpleImportSort,
			"unused-imports": importPlugin,
		},
		rules: {
			"simple-import-sort/imports": [
				"error",
				{
					groups: [
						// Side effect imports.
						["^\\u0000"],
						// builtins
						["^fs$", "^path$", "^child_process$", "^util$"],
						// bun builtins prefixed with `bun:`.
						["^bun:"],
						// node builtins prefixed with `node:`.
						["^node:"],
						// vscode builtins prefixed with `vscode`.
						["^vscode$"],
						// Packages.
						["vscode"],
						// Absolute imports and other imports such as Vue-style `@/foo`.
						["\S"],
						// Relative imports.
						["^\\.", "^@src/"],
					],
				},
			],
			"simple-import-sort/exports": "error",
			"no-unused-vars": "off", // or "@typescript-eslint/no-unused-vars": "off",
			"unused-imports/no-unused-imports": "error",
			"unused-imports/no-unused-vars": [
				"warn",
				{
					vars: "all",
					varsIgnorePattern: "^_",
					args: "after-used",
					argsIgnorePattern: "^_",
				},
			],
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
