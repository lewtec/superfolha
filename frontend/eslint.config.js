import globals from "globals";
import tseslint from "typescript-eslint";
import pluginReactConfig from "eslint-plugin-react/configs/recommended.js";
import tailwind from "eslint-plugin-better-tailwindcss";

const tailwindConfigOptions = {
  entryPoint: "src/index.css",
  tailwindConfig: "tailwind.config.js",
};

const configuredTailwindRules = {};
if (tailwind.configs.recommended.rules) {
  for (const [ruleName, ruleValue] of Object.entries(
    tailwind.configs.recommended.rules,
  )) {
    let level = ruleValue;
    if (Array.isArray(ruleValue)) {
      level = ruleValue[0];
    }
    configuredTailwindRules[ruleName] = [level, tailwindConfigOptions];
  }
}

// Disable no-unknown-classes due to issues with DaisyUI detection
configuredTailwindRules["better-tailwindcss/no-unknown-classes"] = "off";
// Disable consistent-line-wrapping as it conflicts with Prettier
configuredTailwindRules["better-tailwindcss/enforce-consistent-line-wrapping"] =
  "off";

export default [
  {
    ignores: [
      "dist/",
      "tailwind.config.js",
      "postcss.config.js",
      "vite.config.ts",
    ],
  },
  {
    files: ["**/*.{js,mjs,cjs,ts,jsx,tsx}"],
    languageOptions: { globals: globals.browser },
    settings: {
      react: {
        version: "detect",
      },
    },
  },
  ...tseslint.configs.recommended,
  {
    ...pluginReactConfig,
    files: ["**/*.{jsx,tsx}"],
    rules: {
      ...pluginReactConfig.rules,
      "react/react-in-jsx-scope": "off",
      "react/prop-types": "off",
      "react/no-unescaped-entities": "off",
    },
  },
  {
    rules: {
      "@typescript-eslint/no-unused-vars": "warn",
      "@typescript-eslint/no-explicit-any": "warn",
    },
  },
  tailwind.configs.recommended,
  {
    rules: configuredTailwindRules,
  },
];
