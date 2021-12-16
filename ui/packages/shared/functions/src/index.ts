export const capitalize = (a: string): string =>
  a
    .split(' ')
    .map(p => p[0].toUpperCase() + p.substr(1).toLocaleLowerCase())
    .join(' ')

const unitsInTime = [
  { value: 1, symbol: 'ns' },
  { value: 1e3, symbol: 'µs' },
  { value: 1e6, symbol: 'ms' },
  { value: 1e9, symbol: 's' },
  { value: 6 * 1e10, symbol: 'm' }
]

const unitsInBytes = [
  { value: 1, symbol: 'Bytes' },
  { value: 1e3, symbol: 'kB' },
  { value: 1e6, symbol: 'MB' },
  { value: 1e9, symbol: 'GB' },
  { value: 1e12, symbol: 'TB' },
  { value: 1e15, symbol: 'PB' },
  { value: 1e18, symbol: 'EB' }
]

const unitsInCount = [
  { value: 1, symbol: '' },
  { value: 1e3, symbol: 'k' },
  { value: 1e6, symbol: 'M' },
  { value: 1e9, symbol: 'G' },
  { value: 1e12, symbol: 'T' },
  { value: 1e15, symbol: 'P' },
  { value: 1e18, symbol: 'E' }
]

const knownValueFormatters = {
  bytes: unitsInBytes,
  nanoseconds: unitsInTime,
  count: unitsInCount
}

export const valueFormatter = (num: number, unit: string, digits: number): string => {
  const absoluteNum = Math.abs(num)
  const format = knownValueFormatters[unit]

  if (format === undefined || format === null) {
    return num.toString()
  }

  const rx = /\.0+$|(\.[0-9]*[1-9])0+$/
  let i
  for (i = format.length - 1; i > 0; i--) {
    if (absoluteNum >= format[i].value) {
      break
    }
  }
  return (num / format[i].value).toFixed(digits).replace(rx, '$1') + format[i].symbol
}
