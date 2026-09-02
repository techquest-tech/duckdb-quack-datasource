const path = require('path');
const { ProvidePlugin } = require('webpack');

const GRAFANA_EXTERNALS = [
  'react', 'react-dom', '@grafana/schema', '@grafana/data',
  '@grafana/ui', '@grafana/runtime', 'rxjs', 'rxjs/operators',
];

module.exports = {
  mode: 'production',
  entry: './src/module.ts',
  devtool: 'source-map',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: 'module.js',
    library: { type: 'umd' },
    clean: false,
  },
  resolve: { extensions: ['.ts', '.tsx', '.js', '.jsx'] },
  externals: (ctx, request, callback) => {
    if (GRAFANA_EXTERNALS.includes(request)) {
      callback(null, { commonjs: request, commonjs2: request, amd: request, root: request });
      return;
    }
    callback();
  },
  module: {
    rules: [
      { test: /\.tsx?$/, use: { loader: 'ts-loader', options: { transpileOnly: true } }, exclude: /node_modules/ },
      { test: /\.css$/, use: ['style-loader', 'css-loader'] },
    ],
  },
  plugins: [new ProvidePlugin({ React: 'react' })],
};
