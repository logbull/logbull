import { LoadingOutlined, PlusOutlined } from '@ant-design/icons';
import { AutoComplete, DatePicker, Input, Select, Spin, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';

import type { ConditionNode, QueryOperator, QueryableField } from '../../../entity/query';

const { Option } = Select;

interface Props {
  fields: QueryableField[];
  condition?: ConditionNode;
  onChange: (condition: ConditionNode) => void;
  onFieldSearch?: (searchTerm?: string) => Promise<QueryableField[]>;
}

export const ConditionEditorComponent = ({
  fields,
  condition,
  onChange,
  onFieldSearch,
}: Props): React.JSX.Element => {
  // States
  const [arrayValues, setArrayValues] = useState<string[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [localFields, setLocalFields] = useState<QueryableField[]>(fields);
  const [isLocalSearching, setIsLocalSearching] = useState(false);
  const [searchTimeout, setSearchTimeout] = useState<ReturnType<typeof setTimeout> | null>(null);

  // Functions
  const debouncedSearchFields = async (searchTerm?: string) => {
    // Clear existing timeout
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }

    // Set loading state immediately
    setIsLocalSearching(true);

    // Create new timeout
    const timeoutId = setTimeout(async () => {
      if (onFieldSearch && searchTerm && searchTerm.length > 2) {
        try {
          const searchResults = await onFieldSearch(searchTerm);
          setLocalFields(searchResults);
        } catch (error) {
          console.error('Field search failed:', error);
          // Fallback to original fields if search fails
          setLocalFields(fields);
        }
      } else {
        // Reset to original fields if no search term or search term too short
        setLocalFields(fields);
      }
      setIsLocalSearching(false);
    }, 250);

    setSearchTimeout(timeoutId);
  };

  const getOrCreateField = (fieldName: string): QueryableField => {
    const existingField = fields.find((f) => f.name === fieldName);
    if (existingField) {
      return existingField;
    }

    // Create a default field object for unknown fields
    return {
      name: fieldName,
      type: 'string',
      operations: [
        'equals',
        'not_equals',
        'contains',
        'not_contains',
        'exists',
        'not_exists',
      ] as QueryOperator[],
      isCustom: true,
    };
  };

  const getDefaultValueForOperator = (
    operator: QueryOperator,
  ): string | number | boolean | string[] | null => {
    switch (operator) {
      case 'exists':
      case 'not_exists':
        return null;
      case 'in':
      case 'not_in':
        return [];
      default:
        return '';
    }
  };

  const operatorNeedsValue = (operator: QueryOperator): boolean => {
    return operator !== 'exists' && operator !== 'not_exists';
  };

  const operatorExpectsArray = (operator: QueryOperator): boolean => {
    return operator === 'in' || operator === 'not_in';
  };

  const isTimestampField = (fieldName: string): boolean => {
    return fieldName === 'timestamp' || fieldName === 'created_at';
  };

  const handleFieldChange = (fieldName: string) => {
    // Allow empty field names
    if (!fieldName || fieldName.trim() === '') {
      onChange({
        field: '',
        operator: 'equals',
        value: '',
      });
      return;
    }

    const field = getOrCreateField(fieldName);

    // For "message" field, default to "contains" operator since it's more commonly used than "equals"
    const newOperator =
      fieldName === 'message' && field.operations.includes('contains')
        ? 'contains'
        : field.operations[0];

    onChange({
      field: fieldName,
      operator: newOperator,
      value: getDefaultValueForOperator(newOperator),
    });
  };

  const handleOperatorChange = (operator: QueryOperator) => {
    const currentValue = condition?.value;
    let newValue = currentValue;

    // Only reset value if we're switching to/from operators with incompatible value types
    if (!operatorNeedsValue(operator)) {
      // Switching to exists/not_exists - these don't need values
      newValue = null;
    } else if (operatorExpectsArray(operator) && !Array.isArray(currentValue)) {
      // Switching to in/not_in from a single value - convert to array if there's a value
      newValue = currentValue && currentValue !== '' ? [String(currentValue)] : [];
    } else if (!operatorExpectsArray(operator) && Array.isArray(currentValue)) {
      // Switching from in/not_in to a single value operator - use first array element or empty string
      newValue = currentValue.length > 0 ? String(currentValue[0]) : '';
    } else if (newValue === null || newValue === undefined) {
      // Only use default if there's no current value
      newValue = getDefaultValueForOperator(operator);
    }

    onChange({
      field: condition?.field || currentField?.name || '',
      operator,
      value: newValue,
    });
  };

  const handleValueChange = (value: string | number | boolean | string[] | null) => {
    onChange({
      field: condition?.field || currentField?.name || '',
      operator: currentOperator,
      value,
    });
  };

  const handleArrayValueAdd = () => {
    if (inputValue.trim()) {
      const newValues = [...arrayValues, inputValue.trim()];
      setArrayValues(newValues);
      setInputValue('');
      handleValueChange(newValues);
    }
  };

  const handleArrayValueRemove = (index: number) => {
    const newValues = arrayValues.filter((_, i) => i !== index);
    setArrayValues(newValues);
    handleValueChange(newValues);
  };

  const renderValueInput = () => {
    if (!operatorNeedsValue(currentOperator)) {
      return null;
    }

    // Array input for IN/NOT IN operators
    if (operatorExpectsArray(currentOperator)) {
      return (
        <div className="space-y-2">
          <div className="flex space-x-2">
            <Input
              placeholder="Enter value"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onPressEnter={handleArrayValueAdd}
              size="small"
              className="flex-1"
            />
            <button
              type="button"
              onClick={handleArrayValueAdd}
              className="rounded bg-emerald-600 px-2 py-1 text-sm text-white hover:bg-emerald-700"
            >
              <PlusOutlined />
            </button>
          </div>

          {arrayValues.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {arrayValues.map((value, index) => (
                <Tag
                  key={index}
                  closable
                  onClose={() => handleArrayValueRemove(index)}
                  className="border-emerald-200 bg-emerald-50"
                >
                  {value}
                </Tag>
              ))}
            </div>
          )}
        </div>
      );
    }

    // Date picker for timestamp fields and comparison operators
    if (
      isTimestampField(condition?.field || '') &&
      [
        'greater_than',
        'greater_or_equal',
        'less_than',
        'less_or_equal',
        'equals',
        'not_equals',
      ].includes(currentOperator)
    ) {
      return (
        <DatePicker
          showTime
          value={condition?.value ? dayjs(condition.value as string) : null}
          onChange={(date) => {
            handleValueChange(date ? date.toISOString() : '');
          }}
          placeholder="Select date and time"
          size="small"
          className="w-full"
        />
      );
    }

    // Regular text input for other cases
    return (
      <Input
        placeholder="Enter value"
        value={(condition?.value as string) || ''}
        onChange={(e) => handleValueChange(e.target.value)}
        size="small"
      />
    );
  };

  // Calculated values
  const currentField =
    condition?.field && condition.field.trim() !== ''
      ? getOrCreateField(condition.field)
      : getOrCreateField('');
  const currentOperator = condition?.operator || currentField?.operations[0] || 'equals';

  const operatorDisplayNames: Record<QueryOperator, string> = {
    equals: 'equals',
    not_equals: 'not equals',
    contains: 'contains',
    not_contains: 'does not contain',
    in: 'is in',
    not_in: 'is not in',
    greater_than: 'greater than',
    greater_or_equal: 'greater than or equal',
    less_than: 'less than',
    less_or_equal: 'less than or equal',
    exists: 'exists',
    not_exists: 'does not exist',
  };

  const fieldOptions = (() => {
    const options = localFields.map((field) => ({
      value: field.name,
      label: (
        <div className="flex items-center justify-between">
          <span>{field.name}</span>
          <span className="text-xs text-gray-400">{field.type}</span>
        </div>
      ),
    }));

    // If field input is empty, move "message" option to the top
    if (!condition?.field || condition.field.trim() === '') {
      const messageIndex = options.findIndex((option) => option.value === 'message');
      if (messageIndex > 0) {
        const messageOption = options.splice(messageIndex, 1)[0];
        options.unshift(messageOption);
      }
    }

    return options;
  })();

  useEffect(() => {
    // Initialize array values if condition has array value
    if (
      operatorExpectsArray(currentOperator) &&
      Array.isArray(condition?.value) &&
      arrayValues.length === 0
    ) {
      setArrayValues(condition.value.map(String));
    }

    // Clear array values if we're not using an array operator
    if (!operatorExpectsArray(currentOperator) && arrayValues.length > 0) {
      setArrayValues([]);
    }
  }, [currentOperator, condition?.value, arrayValues.length]);

  // Update local fields when parent fields change
  useEffect(() => {
    setLocalFields(fields);
  }, [fields]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (searchTimeout) {
        clearTimeout(searchTimeout);
      }
    };
  }, [searchTimeout]);

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-12 items-center gap-2">
        {/* Field Selection */}
        <div className="col-span-4">
          <label className="mb-1 block text-xs font-medium text-gray-600">Field</label>
          <div className="relative">
            <AutoComplete
              value={condition?.field || ''}
              onChange={handleFieldChange}
              onSearch={debouncedSearchFields}
              options={fieldOptions}
              placeholder="Type or select field name"
              size="small"
              className="w-full"
              allowClear
              backfill={false}
              notFoundContent={null}
              filterOption={false}
            />
            {isLocalSearching && (
              <div className="absolute top-1/2 right-2 -translate-y-1/2">
                <Spin indicator={<LoadingOutlined spin style={{ fontSize: 14 }} />} />
              </div>
            )}
          </div>
        </div>

        {/* Operator Selection */}
        <div className="col-span-3">
          <label className="mb-1 block text-xs font-medium text-gray-600">Operator</label>
          <Select
            value={currentOperator}
            onChange={handleOperatorChange}
            size="small"
            className="w-full"
          >
            {currentField?.operations.map((op) => (
              <Option key={op} value={op}>
                {operatorDisplayNames[op]}
              </Option>
            ))}
          </Select>
        </div>

        {/* Value Input */}
        <div className="col-span-5">
          {operatorNeedsValue(currentOperator) && (
            <>
              <label className="mb-1 block text-xs font-medium text-gray-600">Value</label>
              {renderValueInput()}
            </>
          )}

          {!operatorNeedsValue(currentOperator) && (
            <div className="pt-4 text-xs text-gray-400">No value needed</div>
          )}
        </div>
      </div>

      {/* Field info */}
      <div className="flex items-center justify-between text-xs text-gray-500">
        <div>
          <span className="font-medium">{currentField?.type}</span> field{' '}
          {currentField?.type === 'string' ? '(case-sensitive)' : ''}
        </div>
      </div>
    </div>
  );
};
