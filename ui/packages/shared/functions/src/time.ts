import intervalToDuration from 'date-fns/intervalToDuration';

export const formatForTimespan = (from: number, to: number): string => {
  const duration = intervalToDuration({start: from, end: to});
  if (duration <= {minutes: 61}) {
    return 'H:mm';
  }
  if (duration <= {hours: 13}) {
    return 'H';
  }
  if (duration <= {hours: 25}) {
    return 'H:mm D/M';
  }
  return 'd/M';
};
