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
