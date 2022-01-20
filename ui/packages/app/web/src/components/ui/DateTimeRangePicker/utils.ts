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
}

export type DateUnion = RelativeDate | AbsoluteDate;

export class DateTimeRange {
  from: DateUnion;
  to: DateUnion;

  constructor(from = null, to = null) {
    this.from = from || new RelativeDate(UNITS.HOUR, 1);
    this.to = to || new RelativeDate(UNITS.MINUTE, 0);
  }

  getRangeStringForUI() {
    if (this.from.isRelative() && this.to.isRelative() && (this.to as RelativeDate).value === 0) {
      const from = this.from as RelativeDate;
      return `Last ${from.value} ${from.unit}${from.value > 1 ? 's' : ''}`;
    }
    return `${formatDateStringForUI(this.from)} → ${formatDateStringForUI(this.to)}`;
  }

  getDateForPosition(position: POSITION_TYPE) {
    if (position === POSITIONS.FROM) {
      return this.from;
    }
    return this.to;
  }

  setDateForPosition(date: DateUnion, position: string) {
    console.log('setDateForPosition', date, position);
    if (position === POSITIONS.FROM) {
      this.from = date;
    } else {
      this.to = date;
    }
  }
}

export const formatDateStringForUI = dateString => {
  if (dateString.isRelative()) {
    const {unit, value} = dateString;
    if (value === 0) {
      return 'now';
    }
    return `${value} ${unit}${value > 1 ? 's' : ''} ago`;
  }
  return dateString;
};
