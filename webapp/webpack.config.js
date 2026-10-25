const path = require('path');

module.exports = {
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
                        presets: [
                            ['@babel/preset-env', {targets: 'defaults', useBuiltIns: 'usage', corejs: 3}],
                            '@babel/preset-react',
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
};
