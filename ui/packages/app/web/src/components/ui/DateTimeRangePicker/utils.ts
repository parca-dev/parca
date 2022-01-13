export const formatDateStringForUI = dateString => {
  if (dateString.startsWith('relative:')) {
    return dateString.substring('relative:'.length);
  }
  return dateString;
};

export class DateTimeRange {
  from: string;
  to: string;

  constructor(from = null, to = null) {
    this.from = from || 'relative:1hour ago';
    this.to = to || 'relative:now';
  }

  getRangeStringForUI() {
    if (
      this.from.startsWith('relative:') &&
      this.to.startsWith('relative:') &&
      this.to === 'relative:now'
    ) {
      return `Last ${this.from.substring('relative:'.length).replace(' ago', '')}`;
    }
    return `${formatDateStringForUI(this.from)} → ${formatDateStringForUI(this.to)}`;
  }
}
