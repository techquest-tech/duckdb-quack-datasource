import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface DuckDBQuery extends DataQuery {
  sql: string;
}

export interface DuckDBDataSourceOptions extends DataSourceJsonData {
  endpoint?: string;
  tablePrefix?: string;
  queryTimeoutMS?: number;
}

export interface DuckDBSecureOptions {
  token?: string;
}
