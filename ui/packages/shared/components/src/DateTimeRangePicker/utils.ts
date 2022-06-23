import {convertLocalToUTCDate} from '@parca/functions';

export const UNITS = {
  MINUTE: 'minute',
  HOUR: 'hour',
  DAY: 'day',
};

export const POSITIONS = {
  FROM: 'from',
  TO: 'to',
};

export type UNIT_TYPE = typeof UNITS[keyof typeof UNITS];
export type POSITION_TYPE = typeof POSITIONS[keyof typeof POSITIONS];

interface BaseDate {
  isRelative: () => boolean;
}
export class RelativeDate implements BaseDate {
  isRelative = () => true;
  unit: UNIT_TYPE;
  value: number;

  constructor(unit: UNIT_TYPE, value: number) {
    this.unit = unit;
    this.value = value;
  }
}

export class AbsoluteDate implements BaseDate {
  isRelative = () => false;
  value: Date;
  constructor(value?: Date) {
    this.value = value ?? getDateHoursAgo(1);
  }
}

export type DateUnion = RelativeDate | AbsoluteDate;

export class DateTimeRange {
  from: DateUnion;
  to: DateUnion;

  constructor(from: null | DateUnion = null, to: null | DateUnion = null) {
    this.from = from ?? new RelativeDate(UNITS.MINUTE, 15);
    this.to = to ?? new RelativeDate(UNITS.MINUTE, 0);
  }

  getRangeStringForUI(): String {
    if (this.from.isRelative() && this.to.isRelative() && (this.to as RelativeDate).value === 0) {
      const from = this.from as RelativeDate;
      return `Last ${from.value} ${from.unit}${from.value > 1 ? 's' : ''}`;
    }
    const formattedFrom = formatDateStringForUI(this.from);
    const formattedTo = formatDateStringForUI(this.to).replace(
      `${formattedFrom.split(',')[0]},`,
      ''
    );
    return `${formattedFrom} → ${formattedTo}`;
  }

  getDateForPosition(position: POSITION_TYPE) {
    if (position === POSITIONS.FROM) {
      return this.from;
    }
    return this.to;
  }

  setDateForPosition(date: DateUnion, position: string) {
    if (position === POSITIONS.FROM) {
      this.from = date;
    } else {
      this.to = date;
    }
  }

  getMs(date: DateUnion) {
    if (date.isRelative()) {
      return getRelativeDateMs(date as RelativeDate);
    }
    return (date as AbsoluteDate).value.getTime();
  }

  getFromMs() {
    return this.getMs(this.from);
  }

  getToMs() {
    return this.getMs(this.to);
  }

  getDateStringKey(date: DateUnion) {
    if (date.isRelative()) {
      const relativeDate = date as RelativeDate;
      return `${relativeDate.unit}|${relativeDate.value}`;
    }
    const absoluteDate = date as AbsoluteDate;
    return `${absoluteDate.value.getTime()}`;
  }

  getFromDateStringKey() {
    return this.getDateStringKey(this.from);
  }

  getToDateStringKey() {
    return this.getDateStringKey(this.to);
  }

  getRangeKey() {
    if (this.from.isRelative()) {
      return `relative:${this.getFromDateStringKey()}`;
    }
    return `absolute:${this.getFromDateStringKey()}-${this.getToDateStringKey()}`;
  }

  static fromRangeKey(rangeKey: string | undefined) {
    if (rangeKey === undefined) {
      return new DateTimeRange();
    }
    try {
      const [rangeType, rangeValueKey] = rangeKey.split(':');
      if (rangeType === 'relative') {
        const [unit, value] = rangeValueKey.split('|');
        return new DateTimeRange(
          new RelativeDate(unit, parseInt(value, 10)),
          new RelativeDate(UNITS.MINUTE, 0)
        );
      }
      if (rangeType === 'absolute') {
        const [fromKey, toKey] = rangeValueKey.split('-');
        return new DateTimeRange(
          new AbsoluteDate(new Date(parseInt(fromKey, 10))),
          new AbsoluteDate(new Date(parseInt(toKey, 10)))
        );
      }
      throw new Error('Invalid range key');
    } catch (err) {
      console.error('Error while parsing range key', rangeKey, err);
    }
    return new DateTimeRange();
  }

  static fromAbsoluteDates(from: number, to: number) {
    return new DateTimeRange(new AbsoluteDate(new Date(from)), new AbsoluteDate(new Date(to)));
  }
}

export const formatDateStringForUI: (dateString: DateUnion) => string = dateString => {
  if (dateString.isRelative()) {
    const {unit, value} = dateString as RelativeDate;
    if (value === 0) {
      return 'now';
    }
    return `${value} ${unit}${value > 1 ? 's' : ''} ago`;
  }
  return convertLocalToUTCDate(dateString.value as Date).toLocaleDateString('en-GB', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: 'numeric',
    hour12: true,
  });
};

export const getDateHoursAgo = (hours = 1): Date => {
  const now = new Date();
  now.setHours(now.getHours() - hours);
  return now;
};

const getRelativeDateMs = (date: RelativeDate): number => {
  const now = new Date().getTime();
  const {unit, value} = date;
  switch (unit) {
    case UNITS.MINUTE:
      return now - value * 60 * 1000;
    case UNITS.HOUR:
      return now - value * 60 * 60 * 1000;
    case UNITS.DAY:
      return now - value * 24 * 60 * 60 * 1000;
    default:
      return now;
  }
};
