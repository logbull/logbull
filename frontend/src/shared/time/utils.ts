import dayjs, { Dayjs } from 'dayjs';

// Detect whether the user's locale prefers 12-hour time
export const getUserTimeFormat = () => {
  const locale = navigator.language || 'en-US';
  return /[AP]M/.test(new Date(2023, 0, 1, 13).toLocaleTimeString(locale, { hour: 'numeric' }));
};

// UTC weekday (1–7) + HH:mm → local weekday (1–7)
export const getLocalWeekday = (utcWeekday: number, utcTime: string): number => {
  const [h, m] = utcTime.split(':').map(Number);
  const local = dayjs
    .utc()
    .day(utcWeekday % 7)
    .hour(h)
    .minute(m)
    .local();
  return ((local.day() + 6) % 7) + 1; // 1 = Mon … 7 = Sun
};

// Local weekday (1–7) + local time → UTC weekday (1–7)
export const getUtcWeekday = (localWeekday: number, localTime: Dayjs): number => {
  const utc = dayjs()
    .day(localWeekday % 7)
    .hour(localTime.hour())
    .minute(localTime.minute())
    .utc();
  return ((utc.day() + 6) % 7) + 1;
};

// UTC day-of-month (1–31) + HH:mm → local day-of-month (1–31)
export const getLocalDayOfMonth = (utcDom: number, utcTime: string): number => {
  const [h, m] = utcTime.split(':').map(Number);
  const local = dayjs.utc().date(utcDom).hour(h).minute(m).local();
  return local.date();
};

// Local day-of-month + local time → UTC day-of-month (1–31)
export const getUtcDayOfMonth = (localDom: number, localTime: Dayjs): number => {
  const utc = localTime.date(localDom).utc();
  return utc.date();
};
