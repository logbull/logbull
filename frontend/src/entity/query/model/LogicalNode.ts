import type { LogicalOperator } from './LogicalOperator';
import type { QueryNode } from './QueryNode';

export interface LogicalNode {
  operator: LogicalOperator;
  children: QueryNode[];
}
