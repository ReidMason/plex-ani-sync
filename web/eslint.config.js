import js from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "src/api", "src/components/ui"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    rules: {
      "no-inline-comments": "error",
      "no-warning-comments": ["error", { terms: ["todo", "fixme", "xxx", "hack"] }],
      "no-restricted-syntax": [
        "error",
        {
          selector: "ExportNamedDeclaration[source]",
          message: "Don't re-export; import from the defining module.",
        },
        {
          selector: "ExportAllDeclaration",
          message: "Don't re-export; import from the defining module.",
        },
      ],
    },
  },
);
