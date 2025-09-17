import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Select } from 'antd';
import React from 'react';

import type {
  ConditionNode,
  LogicalOperator,
  QueryNode,
  QueryableField,
} from '../../../entity/query';
import { ConditionEditorComponent } from './ConditionEditorComponent';

interface Props {
  fields: QueryableField[];
  query: QueryNode | null;
  onChange: (query: QueryNode | null) => void;
  onFieldSearch?: (searchTerm?: string) => void;
  isSearchingFields?: boolean;
}

export const QueryBuilderComponent = ({
  fields,
  query,
  onChange,
  onFieldSearch,
  isSearchingFields,
}: Props): React.JSX.Element => {
  const createEmptyCondition = (): QueryNode => ({
    type: 'condition',
    condition: {
      field: '',
      operator: 'equals',
      value: '',
    },
  });

  const createLogicalGroup = (operator: LogicalOperator): QueryNode => ({
    type: 'logical',
    logic: {
      operator,
      children: [createEmptyCondition()],
    },
  });

  const handleAddCondition = () => {
    const newCondition = createEmptyCondition();

    if (!query) {
      onChange(newCondition);
      return;
    }

    // If current query is a single condition, wrap both in an AND group
    if (query.type === 'condition') {
      onChange({
        type: 'logical',
        logic: {
          operator: 'and',
          children: [query, newCondition],
        },
      });
      return;
    }

    // If current query is already logical, add to its children
    if (query.type === 'logical' && query.logic) {
      const updatedQuery = {
        ...query,
        logic: {
          ...query.logic,
          children: [...query.logic.children, newCondition],
        },
      };
      onChange(updatedQuery);
    }
  };

  const handleAddLogicalGroup = (operator: LogicalOperator) => {
    const newGroup = createLogicalGroup(operator);

    if (!query) {
      onChange(newGroup);
      return;
    }

    // Wrap current query and new group in an AND
    onChange({
      type: 'logical',
      logic: {
        operator: 'and',
        children: [query, newGroup],
      },
    });
  };

  const updateNode = (path: number[], updatedNode: QueryNode) => {
    if (!query) return;

    const updateNodeRecursive = (node: QueryNode, currentPath: number[]): QueryNode => {
      if (currentPath.length === 0) {
        return updatedNode;
      }

      if (node.type === 'logical' && node.logic) {
        const [index, ...restPath] = currentPath;
        const updatedChildren = [...node.logic.children];
        updatedChildren[index] = updateNodeRecursive(updatedChildren[index], restPath);

        return {
          ...node,
          logic: {
            ...node.logic,
            children: updatedChildren,
          },
        };
      }

      return node;
    };

    onChange(updateNodeRecursive(query, path));
  };

  const removeNode = (path: number[]) => {
    if (!query || path.length === 0) {
      onChange(null);
      return;
    }

    const removeNodeRecursive = (node: QueryNode, currentPath: number[]): QueryNode | null => {
      if (currentPath.length === 1) {
        if (node.type === 'logical' && node.logic) {
          const index = currentPath[0];
          const updatedChildren = node.logic.children.filter((_, i) => i !== index);

          // If only one child remains, return it directly
          if (updatedChildren.length === 1) {
            return updatedChildren[0];
          }

          // If no children remain, return null
          if (updatedChildren.length === 0) {
            return null;
          }

          return {
            ...node,
            logic: {
              ...node.logic,
              children: updatedChildren,
            },
          };
        }
        return null;
      }

      if (node.type === 'logical' && node.logic) {
        const [index, ...restPath] = currentPath;
        const updatedChildren = [...node.logic.children];
        const updatedChild = removeNodeRecursive(updatedChildren[index], restPath);

        if (updatedChild === null) {
          // Remove the child
          updatedChildren.splice(index, 1);

          // If only one child remains, return it directly
          if (updatedChildren.length === 1) {
            return updatedChildren[0];
          }

          // If no children remain, return null
          if (updatedChildren.length === 0) {
            return null;
          }
        } else {
          updatedChildren[index] = updatedChild;
        }

        return {
          ...node,
          logic: {
            ...node.logic,
            children: updatedChildren,
          },
        };
      }

      return node;
    };

    const result = removeNodeRecursive(query, path);
    onChange(result);
  };

  const renderQueryNode = (node: QueryNode, path: number[] = [], depth = 0): React.ReactElement => {
    const indentClass = depth > 0 ? `ml-${Math.min(depth * 4, 16)}` : '';

    if (node.type === 'condition') {
      return (
        <div
          key={`condition-${path.join('-')}`}
          className={`relative max-w-[800px] ${indentClass}`}
        >
          <div className="flex items-start space-x-2 rounded-lg border border-gray-200 bg-gray-50 p-3">
            <div className="flex-1">
              <ConditionEditorComponent
                fields={fields}
                condition={node.condition}
                onChange={(updatedCondition: ConditionNode) =>
                  updateNode(path, { type: 'condition', condition: updatedCondition })
                }
                onFieldSearch={onFieldSearch}
                isSearchingFields={isSearchingFields}
              />
            </div>

            {path.length > 0 && (
              <Button
                type="text"
                size="small"
                icon={<DeleteOutlined />}
                onClick={() => removeNode(path)}
                danger
                className="flex-shrink-0"
              />
            )}
          </div>
        </div>
      );
    }

    if (node.type === 'logical' && node.logic) {
      return (
        <div
          key={`logical-${path.join('-')}`}
          className={`relative w-full max-w-[820px] ${indentClass} pr-4`}
        >
          <div className="rounded-lg border border-emerald-200 bg-white shadow-sm">
            <div className="border-b border-emerald-200 px-4 py-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Select
                    value={node.logic.operator}
                    onChange={(operator: LogicalOperator) => {
                      const updatedNode = {
                        ...node,
                        logic: { ...node.logic!, operator },
                      };
                      updateNode(path, updatedNode);
                    }}
                    size="small"
                    className="w-20"
                  >
                    <Select.Option value="and">AND</Select.Option>
                    <Select.Option value="or">OR</Select.Option>
                    <Select.Option value="not">NOT</Select.Option>
                  </Select>
                  <span className="text-xs text-gray-500">
                    ({node.logic.children.length} condition
                    {node.logic.children.length !== 1 ? 's' : ''})
                  </span>
                </div>

                {path.length > 0 && (
                  <Button
                    type="text"
                    size="small"
                    icon={<DeleteOutlined />}
                    onClick={() => removeNode(path)}
                    danger
                  />
                )}
              </div>
            </div>
            <div className="p-4">
              <div className="space-y-3">
                {node.logic.children.map((child, index) =>
                  renderQueryNode(child, [...path, index], depth + 1),
                )}

                {/* Add condition button - only show for non-NOT operators or if NOT has no children */}
                {(node.logic.operator !== 'not' || node.logic.children.length === 0) && (
                  <div className="flex justify-start pt-2">
                    <Button
                      type="dashed"
                      size="small"
                      icon={<PlusOutlined />}
                      onClick={() => {
                        const newCondition = createEmptyCondition();
                        const updatedNode = {
                          ...node,
                          logic: {
                            ...node.logic!,
                            children: [...node.logic!.children, newCondition],
                          },
                        };
                        updateNode(path, updatedNode);
                      }}
                      disabled={node.logic.operator === 'not' && node.logic.children.length >= 1}
                    >
                      Add Condition
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      );
    }

    return <div key={`unknown-${path.join('-')}`}>Unknown node type</div>;
  };

  // Always show the query builder, even with no fields - users can enter custom field names

  return (
    <div className="space-y-4">
      {query ? (
        renderQueryNode(query)
      ) : (
        <div className="rounded-lg border-2 border-dashed border-gray-300 py-8 text-center">
          <p className="mb-4 text-gray-500">No query built yet</p>
          <p className="mb-6 text-sm text-gray-400">
            Start by adding a condition or logical group, or execute without a query to see all logs
          </p>
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex justify-center">
        <div className="flex flex-wrap gap-2">
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleAddCondition}
            className="border-emerald-600 bg-emerald-600 hover:border-emerald-700 hover:bg-emerald-700"
          >
            Add Condition
          </Button>

          <Button icon={<PlusOutlined />} onClick={() => handleAddLogicalGroup('and')}>
            Add AND Group
          </Button>

          <Button icon={<PlusOutlined />} onClick={() => handleAddLogicalGroup('or')}>
            Add OR Group
          </Button>

          <Button icon={<PlusOutlined />} onClick={() => handleAddLogicalGroup('not')}>
            Add NOT Group
          </Button>

          {query && (
            <Button danger onClick={() => onChange(null)}>
              Clear All
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};
