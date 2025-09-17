module.exports = {
  root: true,
  env: {
    node: true,
    browser: true,
    es2022: true,
  },
  parser: "vue-eslint-parser",
  parserOptions: {
    parser: "@typescript-eslint/parser",
    ecmaVersion: 2022,
    sourceType: "module",
    extraFileExtensions: [".vue"],
  },
  extends: [
    "eslint:recommended",
    "@vue/eslint-config-typescript/recommended",
    "@vue/eslint-config-prettier",
    "plugin:vue/vue3-recommended",
  ],
  plugins: ["@typescript-eslint", "vue"],
  globals: {
    defineProps: "readonly",
    defineEmits: "readonly",
    defineExpose: "readonly",
    withDefaults: "readonly",
  },
  rules: {
    // Vue specific rules
    "vue/multi-word-component-names": "off",
    "vue/no-v-html": "off",
    "vue/require-default-prop": "off",
    "vue/require-explicit-emits": "off",
    "vue/attribute-hyphenation": "off",
    "vue/v-on-event-hyphenation": "off",

    // TypeScript rules
    "@typescript-eslint/no-unused-vars": [
      "error",
      {
        argsIgnorePattern: "^_",
        varsIgnorePattern: "^_",
      },
    ],
    "@typescript-eslint/no-explicit-any": "warn",
    "@typescript-eslint/no-empty-function": "off",
    "@typescript-eslint/ban-ts-comment": "off",
    "@typescript-eslint/no-non-null-assertion": "off",

    // General rules
    "no-console": process.env.NODE_ENV === "production" ? "warn" : "off",
    "no-debugger": process.env.NODE_ENV === "production" ? "warn" : "off",
    "prefer-const": "error",
    "no-var": "error",
  },
  ignorePatterns: ["dist/", "node_modules/", "*.d.ts"],
};
