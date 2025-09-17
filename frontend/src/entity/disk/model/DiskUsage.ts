import type { Platform } from './Platform';

export type DiskUsage = {
  platform: Platform;
  totalSpaceBytes: number;
  usedSpaceBytes: number;
  freeSpaceBytes: number;
};
