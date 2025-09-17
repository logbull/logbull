import type { ConditionNode } from './ConditionNode';
import type { LogicalNode } from './LogicalNode';

export interface QueryNode {
  type: 'condition' | 'logical';
  condition?: ConditionNode;
  logic?: LogicalNode;
}
