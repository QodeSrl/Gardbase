import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import path from "path";

export default [
  {
    ignores: [
      "**/dist/**",
      "**/node_modules/**",
      "**/public/**",
      "**/vite.config.ts",
      "**/vite.config.*.timestamp*",
      "**/vitest.config.*.timestamp*",
    ],
  },

  // Base JS rules (must be wrapped in an array)
  js.configs.recommended,

  // TypeScript + type-aware rules (monorepo support)
  ...tseslint.configs.recommendedTypeChecked.map(config => ({
    ...config,
    files: ["**/*.ts", "**/*.tsx"],
    languageOptions: {
      ...config.languageOptions,
      parserOptions: {
        ...config.languageOptions?.parserOptions,
        project: ["./tsconfig.json", "./apps/*/tsconfig.json", "./packages/*/tsconfig.json"],
        tsconfigRootDir: path.resolve(),
      },
    },
  })),

  // React-specific rules
  {
    files: ["**/*.tsx"],
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "off",
    },
  },
];
