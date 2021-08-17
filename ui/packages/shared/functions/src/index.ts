export const capitalize = (a: string): string =>
  a
    .split(' ')
    .map(p => p[0].toUpperCase() + p.substr(1).toLocaleLowerCase())
    .join(' ')
