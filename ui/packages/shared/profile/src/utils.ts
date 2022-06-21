export const hexifyAddress = (address?: string): string => {
  if (address == null) {
    return '';
  }
  return `0x${parseInt(address, 10).toString(16)}`;
};
