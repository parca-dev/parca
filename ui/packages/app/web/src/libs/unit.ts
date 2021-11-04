export const nFormatter = (num: number, unit: string, digits: number): string => {
  console.log(unit)
  const formats = unit === 'nanoseconds'
    ? [
        { value: 1, symbol: 'ns' },
        { value: 1E3, symbol: 'µs' },
        { value: 1E6, symbol: 'ms' },
        { value: 1E9, symbol: 's' },
        { value: 6 * 1E10, symbol: 'm' }
      ]
    : [
        { value: 1, symbol: '' },
        { value: 1E3, symbol: 'k' },
        { value: 1E6, symbol: 'M' },
        { value: 1E9, symbol: 'G' },
        { value: 1E12, symbol: 'T' },
        { value: 1E15, symbol: 'P' },
        { value: 1E18, symbol: 'E' }
      ]
  const rx = /\.0+$|(\.[0-9]*[1-9])0+$/
  let i
  for (i = formats.length - 1; i > 0; i--) {
    if (num >= formats[i].value) {
      break
    }
  }
  return (num / formats[i].value).toFixed(digits).replace(rx, '$1') + formats[i].symbol
}
