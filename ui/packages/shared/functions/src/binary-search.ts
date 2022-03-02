export function binarySearchClosest(sortedArray: number[], seekElement: number): number {
  if (sortedArray.length === 1) {
    return 0;
  }

  let startIndex = 0;
  let endIndex: number = sortedArray.length - 1;
  while (startIndex <= endIndex) {
    if (endIndex - startIndex === 1) {
      const distanceToStart = seekElement - sortedArray[startIndex];
      const distanceToEnd = sortedArray[endIndex] - seekElement;
      if (distanceToStart < distanceToEnd) {
        return startIndex;
      }
      if (distanceToStart > distanceToEnd) {
        return endIndex;
      }
    }
    const mid = startIndex + Math.floor((endIndex - startIndex) / 2);
    const guess = sortedArray[mid];
    if (guess > seekElement) {
      endIndex = mid;
    } else {
      startIndex = mid;
    }
  }

  return -1;
}
