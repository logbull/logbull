export const getUserTimeFormat = () => {
  const locale = navigator.language || 'en-US';
  const testDate = new Date(2023, 0, 1, 13, 0, 0); // 1 PM
  const timeString = testDate.toLocaleTimeString(locale, { hour: 'numeric' });
  const is12Hour = timeString.includes('PM') || timeString.includes('AM');

  return {
    use12Hours: is12Hour,
    format: is12Hour ? 'DD.MM.YYYY h:mm:ss A' : 'DD.MM.YYYY HH:mm:ss',
  };
};

export const getUserShortTimeFormat = () => {
  const locale = navigator.language || 'en-US';
  const testDate = new Date(2023, 0, 1, 13, 0, 0); // 1 PM
  const timeString = testDate.toLocaleTimeString(locale, { hour: 'numeric' });
  const is12Hour = timeString.includes('PM') || timeString.includes('AM');

  return {
    use12Hours: is12Hour,
    format: is12Hour ? 'DD MMM YYYY h:mm A' : 'DD MMM YYYY HH:mm',
  };
};
