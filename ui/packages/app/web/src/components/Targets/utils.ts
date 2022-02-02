import moment from 'moment';

export enum HealthStatus {
  'Unspecified',
  'Good',
  'Bad',
}

export const getHealthStatus = (numericValue: number) => {
  const label = HealthStatus[numericValue];
  const colorVariants = {
    Unspecified: 'neutral',
    Good: 'success',
    Bad: 'danger',
  };
  return {label, colorVariant: colorVariants[label]};
};

export const secondsInNanos = (seconds: number) => seconds * 1e9;
export const millisecondsInNanos = (milli: number) => milli * 1e3;
export const nanosInMilliseconds = (nanos: number) => nanos / 1e6;
export const formatToMilliseconds = ({
  seconds = 0,
  milliseconds = 0,
  nanos = 0,
}: {
  seconds?: number;
  milliseconds?: number;
  nanos?: number;
}) => nanosInMilliseconds(secondsInNanos(seconds) + millisecondsInNanos(milliseconds) + nanos);

// functions below taken from Prometheus, with slight modifications
// https://github.com/prometheus/prometheus/blob/main/web/ui/react-app/src/utils/index.ts#L82-L112
export const formatDuration = (milliseconds: number): string => {
  let ms = milliseconds;
  let r = '';
  if (ms === 0) {
    return '0s';
  }

  const f = (unit: string, mult: number, exact: boolean) => {
    if (exact && ms % mult !== 0) {
      return;
    }
    const v = Math.floor(ms / mult);
    if (v > 0) {
      r += `${v}${unit}`;
      ms -= v * mult;
    }
  };

  // Only format years and weeks if the remainder is zero, as it is often
  // easier to read 90d than 12w6d.
  f('y', 1000 * 60 * 60 * 24 * 365, true);
  f('w', 1000 * 60 * 60 * 24 * 7, true);

  f('d', 1000 * 60 * 60 * 24, false);
  f('h', 1000 * 60 * 60, false);
  f('m', 1000 * 60, false);
  f('s', 1000, false);
  f('ms', 1, false);

  return r;
};

export const humanizeDuration = (milliseconds: number): string => {
  const sign = milliseconds < 0 ? '-' : '';
  const unsignedMillis = milliseconds < 0 ? -1 * milliseconds : milliseconds;
  const duration = moment.duration(unsignedMillis, 'ms');
  const ms = Math.floor(duration.milliseconds());
  const s = Math.floor(duration.seconds());
  const m = Math.floor(duration.minutes());
  const h = Math.floor(duration.hours());
  const d = Math.floor(duration.asDays());
  if (d !== 0) {
    return `${sign}${d}d ${h}h ${m}m ${s}s`;
  }
  if (h !== 0) {
    return `${sign}${h}h ${m}m ${s}s`;
  }
  if (m !== 0) {
    return `${sign}${m}m ${s}s`;
  }
  if (s !== 0) {
    return `${sign}${s}.${ms}s`;
  }
  if (unsignedMillis > 0) {
    return `${sign}${unsignedMillis.toFixed(3)}ms`;
  }
  return '0s';
};

export const now = (): number => moment().valueOf();

export const formatRelative = (start: number, end: number): string => {
  if (!start) {
    return 'Never';
  }

  if (!end) {
    humanizeDuration(now() - start) + ' ago';
  }

  return humanizeDuration(end - start) + ' ago';
};
