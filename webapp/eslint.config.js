const js = require('@eslint/js');
const react = require('eslint-plugin-react');
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
        plugins: {react},
        rules: {'react/jsx-uses-react': 'error', 'react/jsx-uses-vars': 'error'},
    },
];
