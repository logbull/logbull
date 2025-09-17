import { ClockCircleOutlined } from '@ant-design/icons';
import { DatePicker, Select } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';

const { RangePicker } = DatePicker;

export interface TimeRange {
  from: dayjs.Dayjs;
  to: dayjs.Dayjs;
}

export interface TimeRangePreset {
  label: string;
  value: string;
  getRange: () => TimeRange;
}

const presets: TimeRangePreset[] = [
  {
    label: 'Last 5 minutes',
    value: '5m',
    getRange: () => ({
      from: dayjs().subtract(5, 'minutes'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 15 minutes',
    value: '15m',
    getRange: () => ({
      from: dayjs().subtract(15, 'minutes'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 30 minutes',
    value: '30m',
    getRange: () => ({
      from: dayjs().subtract(30, 'minutes'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 1 hour',
    value: '1h',
    getRange: () => ({
      from: dayjs().subtract(1, 'hour'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 4 hours',
    value: '4h',
    getRange: () => ({
      from: dayjs().subtract(4, 'hours'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 12 hours',
    value: '12h',
    getRange: () => ({
      from: dayjs().subtract(12, 'hours'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 24 hours',
    value: '24h',
    getRange: () => ({
      from: dayjs().subtract(24, 'hours'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 7 days',
    value: '7d',
    getRange: () => ({
      from: dayjs().subtract(7, 'days'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 1 month',
    value: '1m',
    getRange: () => ({
      from: dayjs().subtract(1, 'month'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 3 months',
    value: '3m',
    getRange: () => ({
      from: dayjs().subtract(3, 'months'),
      to: dayjs(),
    }),
  },
  {
    label: 'Last 1 year',
    value: '1y',
    getRange: () => ({
      from: dayjs().subtract(1, 'year'),
      to: dayjs(),
    }),
  },
];

interface Props {
  onChange: (range: TimeRange | null) => void;
  onGetCurrentRange?: (getCurrentRange: () => TimeRange | null) => void;
  onGetRangeHelpers?: (helpers: { isUntilNow: () => boolean; refreshRange: () => void }) => void;
}

export const TimeRangePickerComponent = ({
  onChange,
  onGetCurrentRange,
  onGetRangeHelpers,
}: Props): React.JSX.Element => {
  // States
  const [selectedPreset, setSelectedPreset] = useState<string>('24h');
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);

  // Functions
  const getCurrentRange = (): TimeRange | null => {
    if (selectedPreset === 'custom') {
      return customRange ? { from: customRange[0], to: customRange[1] } : null;
    }

    const preset = presets.find((p) => p.value === selectedPreset);
    return preset ? preset.getRange() : null;
  };

  const isUntilNow = (): boolean => {
    // Only presets (not custom) can be "until now"
    return selectedPreset !== 'custom';
  };

  const refreshRange = (): void => {
    if (selectedPreset !== 'custom') {
      // For presets, recalculate the range (which will update "now")
      const preset = presets.find((p) => p.value === selectedPreset);
      if (preset) {
        const range = preset.getRange();
        onChange(range);
      }
    }
  };

  const handlePresetChange = (presetValue: string) => {
    setSelectedPreset(presetValue);

    if (presetValue === 'custom') {
      // Keep custom range if available, otherwise notify parent with null
      const range = customRange ? { from: customRange[0], to: customRange[1] } : null;
      onChange(range);
    } else {
      // Calculate and notify parent with preset range
      const preset = presets.find((p) => p.value === presetValue);
      if (preset) {
        const range = preset.getRange();
        onChange(range);
      }
    }
  };

  const handleCustomRangeChange = (dates: [dayjs.Dayjs, dayjs.Dayjs] | null) => {
    setCustomRange(dates);

    if (selectedPreset === 'custom') {
      const range = dates ? { from: dates[0], to: dates[1] } : null;
      onChange(range);
    }
  };

  // useEffect hooks
  useEffect(() => {
    const defaultPreset = presets.find((p) => p.value === selectedPreset);
    if (defaultPreset) {
      const range = defaultPreset.getRange();
      onChange(range);
    }
  }, []);

  useEffect(() => {
    if (onGetCurrentRange) {
      onGetCurrentRange(getCurrentRange);
    }
  }, [selectedPreset, customRange, onGetCurrentRange]);

  useEffect(() => {
    if (onGetRangeHelpers) {
      onGetRangeHelpers({
        isUntilNow,
        refreshRange,
      });
    }
  }, [selectedPreset, customRange, onGetRangeHelpers]);

  return (
    <div className="space-y-3">
      <div>
        <label className="mb-1 block text-sm font-medium text-gray-700">Time Range</label>
        <Select
          value={selectedPreset}
          onChange={handlePresetChange}
          className="w-48"
          suffixIcon={<ClockCircleOutlined />}
        >
          <Select.Option value="custom">Custom Range</Select.Option>

          {presets.map((preset) => (
            <Select.Option key={preset.value} value={preset.value}>
              {preset.label}
            </Select.Option>
          ))}
        </Select>
      </div>

      {selectedPreset === 'custom' && (
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">
            Select Custom Time Range
          </label>
          <RangePicker
            showTime
            value={customRange}
            onChange={(dates) => {
              const validDates =
                dates && dates[0] && dates[1]
                  ? ([dates[0], dates[1]] as [dayjs.Dayjs, dayjs.Dayjs])
                  : null;
              handleCustomRangeChange(validDates);
            }}
            placeholder={['Start time', 'End time']}
            className="w-96"
          />
        </div>
      )}
    </div>
  );
};
