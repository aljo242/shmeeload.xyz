import js from "@eslint/js";
import globals from "globals";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist/**"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["src/**/*.ts"],
    languageOptions: {
      globals: { ...globals.browser },
    },
    rules: {
      "prefer-const": "error",
      eqeqeq: "error",
      "@typescript-eslint/no-non-null-assertion": "error",
    },
  },
);
