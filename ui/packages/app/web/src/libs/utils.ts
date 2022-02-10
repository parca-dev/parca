export const startsWith = (str: string, prefix: string): boolean =>
  str.lastIndexOf(prefix, 0) === 0;

export const cutToMaxStringLength = (input: string, len: number): string => {
  if (input.length <= len) {
    return input;
  }
  return `${input.substring(0, len)}...`;
};
