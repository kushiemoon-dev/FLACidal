import js from '@eslint/js';
import ts from 'typescript-eslint';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';

export default ts.config(
  { ignores: ['dist/**', 'node_modules/**', 'wailsjs/**', 'playwright-report/**', 'test-results/**'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs['flat/recommended'],
  {
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
  },
  {
    rules: {
      // The codebase runs with tsconfig strict:false and uses `any` for Wails/REST payloads.
      '@typescript-eslint/no-explicit-any': 'off',
      // Existing lists are unkeyed; keying each block risks behaviour changes, tracked separately.
      'svelte/require-each-key': 'off',
      // Long placeholders need a mustache to carry "\n" escapes.
      'svelte/no-useless-mustaches': 'off',
      // State is replaced immutably (new Set(...)), so SvelteSet is not needed.
      'svelte/prefer-svelte-reactivity': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { args: 'none', caughtErrors: 'none' }],
    },
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts'],
    languageOptions: { parserOptions: { parser: ts.parser } },
  },
);
