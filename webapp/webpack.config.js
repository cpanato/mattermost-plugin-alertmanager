var path = require('path');
const webpack = require('webpack');

module.exports = {
    entry: ['./src/index.jsx'],
    resolve: {
        modules: ['src', 'node_modules', path.resolve(__dirname)],
        extensions: ['*', '.js', '.jsx', '.tsx'],
        fallback: {
            crypto: require.resolve('crypto-browserify'),
            stream: require.resolve('stream-browserify'),
            buffer: require.resolve('buffer/'),
        }
    },
    plugins: [
        // Work around for Buffer is undefined:
        // https://github.com/webpack/changelog-v5/issues/10
        new webpack.ProvidePlugin({
            Buffer: ['buffer', 'Buffer'],
        }),
        new webpack.ProvidePlugin({
            process: 'process/browser',
        }),
    ],
    optimization: { minimize: false },
    module: {
        rules: [
            {
                test: /\.(js|jsx|ts|tsx)?$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        plugins: [
                            '@babel/plugin-proposal-class-properties',
                            '@babel/plugin-syntax-dynamic-import',
                            '@babel/plugin-proposal-object-rest-spread',
                        ],
                        presets: [
                            [
                                '@babel/preset-env',
                                {
                                    targets: {
                                        chrome: 66,
                                        firefox: 60,
                                        edge: 42,
                                        safari: 12,
                                    },
                                    corejs: 3,
                                    modules: false,
                                    debug: false,
                                    useBuiltIns: 'usage',
                                    shippedProposals: true,
                                },
                            ],
                            [
                                '@babel/preset-react',
                                {
                                    useBuiltIns: true,
                                },
                            ],
                            ['@babel/preset-typescript', {allowNamespaces: true}],
                        ],
                    },
                },
            },
            {
                test: /\.css$/i, 
                use: [ 'style-loader', 'css-loader' ] 
            },
        ],
    },
    externals: {
        react: 'React',
        redux: 'Redux',
        'react-redux': 'ReactRedux',
        'prop-types': 'PropTypes',
        'react-bootstrap': 'ReactBootstrap',
    },
    output: {
        path: path.join(__dirname, '/dist'),
        publicPath: '/',
        filename: 'main.js',
    },
};
