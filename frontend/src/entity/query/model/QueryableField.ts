import type { QueryOperator } from './QueryOperator';

export interface QueryableField {
  name: string;
  type: string;
  operations: QueryOperator[];
  isCustom: boolean;
}
