import moment from 'moment';

export const formatForTimespan = (from: number, to: number): string => {
  const duration = moment.duration(moment(to).diff(from));
  if (duration <= moment.duration(61, 'minutes')) {
    return 'H:mm';
  }
  if (duration <= moment.duration(13, 'hours')) {
    return 'H';
  }
  if (duration <= moment.duration(25, 'hours')) {
    return 'H:mm D/M';
  }
  return 'D/M';
};
