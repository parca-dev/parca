import React from 'react';
import {formatDuration, formatRelative, formatToMilliseconds, now} from './utils';

const LastScrapeCell = ({key, lastScrape, lastScrapeDuration}) => {
  const {seconds: scrapeSeconds, nanos: scrapeNanos} = lastScrape;
  const {seconds: durationSeconds, nanos: durationNanos} = lastScrapeDuration;
  const startInMilliseconds = formatToMilliseconds({seconds: scrapeSeconds, nanos: scrapeNanos});
  const endInMilliseconds = now();
  const durationMilliseconds = formatToMilliseconds({
    seconds: durationSeconds,
    nanos: durationNanos,
  });

  return (
    <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
      <p>Last Scrape: {formatRelative(startInMilliseconds, endInMilliseconds)}</p>
      <p>Duration: {formatDuration(durationMilliseconds)}</p>
    </td>
  );
};

export default LastScrapeCell;
