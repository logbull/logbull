import type { QueryOperator } from './QueryOperator';

export interface ConditionNode {
  field: string;
  operator: QueryOperator;
  value: string | number | boolean | string[] | null;
}
