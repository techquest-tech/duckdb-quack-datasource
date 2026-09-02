import { DataSourcePlugin } from '@grafana/data';

import { DuckDBDatasource } from './datasource';
import { ConfigEditor } from './ConfigEditor';
import { QueryEditor } from './QueryEditor';
import {
  DuckDBDataSourceOptions,
  DuckDBQuery,
  DuckDBSecureOptions,
} from './types';

export const plugin = new DataSourcePlugin<
  DuckDBDatasource,
  DuckDBQuery,
  DuckDBDataSourceOptions,
  DuckDBSecureOptions
>(DuckDBDatasource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
