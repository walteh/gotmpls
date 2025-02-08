import * as pluginJs from "@eslint/js";
import * as tsPlugin from "@typescript-eslint/eslint-plugin";

console.log("TypeScript ESLint configs:", Object.keys(tsPlugin.configs));
console.log("JS ESLint configs:", pluginJs.configs);
