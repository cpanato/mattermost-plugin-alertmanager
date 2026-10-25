const path = require('path');

module.exports = (_env, argv) => ({
    entry: './src/index.jsx',
    resolve: {extensions: ['.js', '.jsx', '.ts', '.tsx']},
    module: {
        rules: [
            {
                test: /\.[jt]sx?$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        envName: argv.mode === 'production' ? 'production' : 'development',
                        targets: 'defaults',
                        plugins: [
                            ['babel-plugin-polyfill-corejs3', {
                                method: 'usage-global',
                                version: require('core-js/package.json').version,
                            }],
                        ],
                        presets: [
                            '@babel/preset-env',
                            ['@babel/preset-react', {runtime: 'classic'}],
                            '@babel/preset-typescript',
                        ],
                    },
                },
            },
            {test: /\.css$/i, use: ['style-loader', 'css-loader']},
        ],
    },
    externals: {
        react: 'React',
        'prop-types': 'PropTypes',
    },
    output: {
        path: path.join(__dirname, 'dist'),
        filename: 'main.js',
        clean: true,
    },
});
