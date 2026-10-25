const js = require('@eslint/js');
const globals = require('globals');

module.exports = [
    {ignores: ['node_modules/**', 'dist/**']},
    js.configs.recommended,
    {
        files: ['**/*.{js,jsx}'],
        languageOptions: {
            ecmaVersion: 'latest',
            sourceType: 'module',
            parserOptions: {ecmaFeatures: {jsx: true}},
            globals: {...globals.browser, ...globals.node},
        },
        // ESLint 10 tracks JSX references; classic JSX still needs the React import.
        rules: {'no-unused-vars': ['error', {varsIgnorePattern: '^React$'}]},
    },
];
